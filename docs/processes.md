# Supervised processes

Use `proc` for a long-running service that should restart after a panic in its
background work, such as a configuration loader or cache refresher.
A process is a mutable Go struct. Its `Run` method runs in a goroutine managed
by a supervisor. Keep a `PID` to call the process; the PID still works after a
restart replaces the struct.

Method calls are ordinary Go calls in the caller's goroutine. There is no
mailbox and calls are not serialized. Use a mutex when methods and `Run` access
the same mutable fields. The checker requires immutable start arguments and
exported method parameters/results. The one exception is an
[unchecked dependency](#unchecked-dependencies) in the start arguments.

## Start and call a process

This counter adds its start argument on each call. `Run` has no background work
in this example; it waits for shutdown. The mutex lets several callers use Add
at the same time.

```go
package main

import (
    "context"
    "fmt"
    "sync"

    "github.com/lucasavila00/immutago/pkg/proc"
)

type Counter struct {
    proc.Process[int]
    mu sync.Mutex
    count int
}

func (*Counter) Run(ctx context.Context) { <-ctx.Done() }

func (c *Counter) Add() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count += c.Args()
    return c.count
}

func main() {
    root := proc.NewRoot()
    counter := proc.Start[Counter](root, 1)
    fmt.Println(counter.Proc().Add()) // 1
    <-root.Stop()
}
```

- `Process[int]` declares the start argument type. `Args()` reads the value `1`
  supplied to Start.
- `Start[Counter]` creates a Counter and starts Run. Keep the returned PID.
- `counter.Proc().Add()` calls Add on the current Counter. It prints `1`.
- `<-root.Stop()` cancels Run's context and waits for it to return.

Check this program with `immutago ./...`, then run it with `go run .`.
See [Get started](getting-started.md) for installation and CI commands.

## Pass data to readers

Start arguments and exported method parameters/results use the
[allowed types](common-forms.md#markers-and-stored-types): numbers, strings,
marked structs, frozen collections, and PIDs. For example, return a count as
`int`; return a list as `im.FrozenSlice[T]`, not `[]T`.

To replace a configuration while readers use the previous version, put an
`im.Owner[Config]` field in the process. Construct each Config as a Scoped
object and store it in the Owner. A method loads it, reads the needed fields,
and returns the result. The [Owner guide](owner.md) shows this sequence.

The [config loader example](../tests/testdata/accepted/process/config_loader.go)
builds and publishes versions on a timer. The
[limiter example](../tests/testdata/accepted/process/limiter.go) receives that
loader's PID in its start arguments and calls it from an exported method.

With those example types, start the loader and limiter in one group:

```go
group := proc.StartSupervisor(root)
loader := proc.Start[ConfigLoader](group, LoaderArgs{Path: "api", Every: time.Minute})
limiter := proc.Start[Limiter](group, LimiterArgs{Loader: loader, Key: "api"})
decision := limiter.Proc().Take()
```

The limiter calls the loader through its PID. Until the loader first publishes,
its limit is zero and Take returns Limited. Both processes stop when you wait
on `<-group.Stop()`.

## Unchecked dependencies

A process can need a logger, an HTTP client, a factory function, or a test
fake. These values are mutable, so they are not allowed types. Wrap each one in
`proc.UncheckedDependency[T]` with `proc.Trust`:

```go
type FetcherArgs struct {
    im.Shared
    BaseURL string
    Log     proc.UncheckedDependency[*slog.Logger]
    Client  proc.UncheckedDependency[*http.Client]
}

func (f *Fetcher) Run(ctx context.Context) {
    f.Args().Log.Value().Info("start", "url", f.Args().BaseURL)
}

proc.Start[Fetcher](root, FetcherArgs{
    BaseURL: "https://example.com",
    Log:     proc.Trust(logger),
    Client:  proc.Trust(http.DefaultClient),
})
```

The checker does not look inside the dependency. With `proc.Trust(v)`, you
promise that concurrent use of `v` is safe. The call makes that promise visible
to reviewers; it does not make `v` immutable. `proc.UncheckedDependency` is the
only exception to the immutable start-argument rules. Code that creates `v`
and uses the result of `Value()` still follows the normal source rules.

A false promise is a data race that the checker cannot find. Without
`proc.Trust`, the checker rejects a map in the start arguments. With it, the
checker accepts the map, as in the
[trusted map](../tests/testdata/accepted/process/trusted_map.go) example:

```go
proc.Start[Reporter](children, ReporterArgs{Seen: proc.Trust(c.seen)}) // accepted; c.seen is still written: a data race
```

Wrap only values that are safe for concurrent use, for example `*slog.Logger`,
`*http.Client`, or a function that does not change shared memory. The
[dependencies](../tests/testdata/accepted/process/dependencies.go) example
shows a logger, a client, a factory function, and a test fake. Read the value
only with `l.Args().Log.Value()` in a process method. The
[process reference](process-forms.md#unchecked-dependencies) lists every form.

## Start and readiness

Start publishes the new struct before Run begins. A method can therefore run
before Run initializes any fields. Put required initial data in the start
arguments, and make methods handle the zero values of other fields.

For two processes that need each other's PID, Reserve one PID, start the first
process, then call StartReserved for the reserved PID. Reserve does not wait
for startup. Calling Proc before StartReserved panics; coordinate the first
calls so that both processes have started.

## Restart and failure

If Run returns, it is not restarted, and its supervisor removes it. If Run panics, the supervisor waits before
restarting it: 1 second after the first panic, then twice as long for each
restart in the last 5 minutes. It creates a new struct from the same start
arguments and replaces the value behind the PID. Mutable fields return to their
zero values, including empty Owner fields.

During the delay, PID calls still use the old struct. A call already in progress
finishes on that old struct after the replacement. A method panic propagates in the caller's goroutine. The supervisor handles
it only if it leaves a supervised Run. The restart rules apply to the process
whose Run failed.

A sixth panic from the same child in 5 minutes stops its siblings, last started
first, and passes the failure to the parent supervisor. The parent restarts the
subtree with the same retry rule. If the failure reaches the root, the program
panics with the last panic value.

## Stop

`root.Stop()` returns a channel. Wait on it to wait for the supervised work to
stop. Children stop in reverse start order. Run must return when its context
is cancelled; Stop cannot force it to exit. Stop does not wait for direct method
calls. A PID can still call methods on its last struct after Run ends or Stop
completes.

Use `proc.StartSupervisor(parent)` to stop a group together and restart that
group after a child exceeds the restart limit. In tests, [proctest.NewRoot(t)](../pkg/proc/proctest/readme.md) stops the
tree during cleanup and fails if it does not stop within 5 seconds.

## Start children

A process can start children under the supervisor of its current run. Pass
`p.Supervisor()` directly to `proc.Start`, `proc.StartReserved`, or
`proc.StartSupervisor`, from `Run` or from any other method:

```go
func (p *WorkerPool) Run(ctx context.Context) {
    sup := p.Supervisor()
    for i := range p.Args().Workers {
        proc.Start[Worker](sup, WorkerArgs{ID: i})
    }
    <-ctx.Done()
}
```

- In a method, `p.Supervisor()` returns the supervisor of the current run of
  that process value. The first call for the value creates it. Children that a
  method starts before `Run` begins start with the run.
- When the process stops, the runtime stops the children of this supervisor,
  last started first, and waits for them. Then it ends the `Run` context, so a
  child can call methods of its parent while the child stops. When `Run`
  returns or panics, the runtime stops the children and waits for them. Then
  the run ends: the restart delay starts, or `Stop` continues.
- After the run ends, `p.Supervisor()` returns a stopped supervisor. `Start`
  under it, or under a stored nested supervisor of that run, returns a PID
  whose `Run` never runs.
- If a child passes its restart limit, the run context ends and the run fails
  with the panic value of the child. The supervisor of the process counts this
  failure as a panic of the process. A group in a nested supervisor first
  restarts alone; if the group fails six times in 5 minutes, the run fails.
- A new run uses a new process value, so it starts with no children and empty
  fields. Children that methods started are gone too. `Run` must start them
  again from data that survives the restart: the start arguments, or a peer
  process, for example a config process with an `im.Owner`, whose PID is in the
  arguments. The [dispatcher](../tests/testdata/accepted/process/dispatcher.go)
  asks its config peer for the current variants in `Run`.
- After a restart of the parent, a PID of an old child keeps its old value, but
  that child does not run again. Do not publish child PIDs: do not store them
  in an `im.Owner`, send them to other processes, or return them from methods.
  Reach a child through its parent on each call.
- To start and stop single children during one run, give each child its own
  nested supervisor: `sub := proc.StartSupervisor(p.Supervisor())`, then
  `proc.Start[Worker](sub, args)`. Keep `sub` in a field, and call
  `<-sub.Stop()` from any method to stop that child. The stopped nested
  supervisor is removed from the supervisor of the run. A child must not
  synchronously call a parent method that stops the group of that child:
  `Stop` waits for the child, and the child waits for the call.

The [process reference](process-forms.md#start-children-under-the-supervisor-of-the-run)
lists the allowed forms.

## Not checked

These checks do not isolate ordinary mutable state. Two processes can still
access the same global map or share data through ordinary helper functions.
Use locks for that data, and run tests with `go test -race`. The race detector
can find a race only when the test executes the conflicting accesses.

The checker also does not check the lifetime of PIDs. A process can reserve a
PID outside its run and start it under the supervisor of its run:

```go
type PoolArgs struct {
    im.Shared
    Spare proc.PID[Worker] // reserved by the caller
}

func (p *Pool) Run(ctx context.Context) {
    proc.StartReserved(p.Supervisor(), p.Args().Spare, WorkerArgs{})
    <-ctx.Done()
}
```

After a restart of `Pool`, the new run calls `StartReserved` on a PID that is
already started. It panics with `proc: PID already started` on each run, and
after six panics in 5 minutes the panic escalates to the parent supervisor. Call
`proc.Reserve` in the same method as `StartReserved`, as the
[dispatcher](../tests/testdata/accepted/process/dispatcher.go) does.

## Reference

The [process reference](process-forms.md) lists every supported declaration and
operation, followed by rejected examples. The [API reference](../pkg/proc/readme.md)
contains function signatures. See [performance](performance.md) for operation costs.
