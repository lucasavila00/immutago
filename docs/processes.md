# Supervised processes

Use `proc` for a long-running service that should restart after a panic in its
background work, such as a configuration loader or cache refresher.
A process is a mutable Go struct. Its `Run` method runs in a goroutine managed
by a supervisor. Keep a `PID` to call the process; the PID still works after a
restart replaces the struct.

Method calls are ordinary Go calls in the caller's goroutine. There is no
mailbox and calls are not serialized. Use a mutex when methods and `Run` access
the same mutable fields. The checker requires immutable start arguments and
parameters/results for methods called through a PID. The one exception is an
[unchecked dependency](#concurrent-dependencies) in the start arguments.

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
    root := proc.NewRoot(nil)
    counter, err := proc.Start[Counter](root, 1)
    if err != nil { panic(err) }
    fmt.Println(counter.Proc().Add()) // 1
    <-root.Stop()
}
```

- `Process[int]` declares the start argument type. `Args()` reads the value `1`
  supplied to Start.
- `Start[Counter]` initializes a Counter and starts Run. Check the error, then keep the PID.
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
loader, err := proc.Start[ConfigLoader](group, LoaderArgs{Path: "api", Every: time.Minute})
if err != nil { panic(err) }
limiter, err := proc.Start[Limiter](group, LimiterArgs{Loader: loader, Key: "api"})
if err != nil { panic(err) }
decision := limiter.Proc().Take()
```

The limiter calls the loader through its PID. Until the loader first publishes,
its limit is zero and Take returns Limited. Both processes stop when you wait
on `<-group.Stop()`.

## Concurrent dependencies

A process can need a logger, an HTTP client, a factory function, or a test
fake. These values are mutable, so they are not allowed types. Wrap each one in
`proc.ConcurrentSafe[T]` with `proc.AssumeConcurrentSafe`:

```go
type FetcherArgs struct {
    im.Shared
    BaseURL string
    Log     proc.ConcurrentSafe[*slog.Logger]
    Client  proc.ConcurrentSafe[*http.Client]
}

func (f *Fetcher) Run(ctx context.Context) {
    f.Args().Log.Value().Info("start", "url", f.Args().BaseURL)
}

proc.Start[Fetcher](root, FetcherArgs{
    BaseURL: "https://example.com",
    Log:     proc.AssumeConcurrentSafe(logger),
    Client:  proc.AssumeConcurrentSafe(http.DefaultClient),
})
```

The checker does not look inside the dependency. With `proc.AssumeConcurrentSafe(v)`, you
promise that concurrent use of `v` is safe. The call makes that promise visible
to reviewers; it does not make `v` immutable. `proc.ConcurrentSafe` is the
only exception to the immutable start-argument rules. Code that creates `v`
and uses the result of `Value()` still follows the normal source rules.

A false promise is a data race that the checker cannot find. Without
`proc.AssumeConcurrentSafe`, the checker rejects a map in the start arguments. With it, the
checker accepts the map, as in the
[trusted map](../tests/testdata/accepted/process/trusted_map.go) example:

```go
proc.Start[Reporter](children, ReporterArgs{Seen: proc.AssumeConcurrentSafe(c.seen)}) // accepted; c.seen is still written: a data race
```

Wrap only values that are safe for concurrent use, for example `*slog.Logger`,
`*http.Client`, or a function that does not change shared memory. The
[dependencies](../tests/testdata/accepted/process/dependencies.go) example
shows a logger, a client, a factory function, and a test fake. Read the value
only with `l.Args().Log.Value()` in a process method. The
[process reference](process-forms.md#concurrent-dependencies) lists every form.

## Set up state before calls

Declare `Init() error` when methods need state built from the start arguments:

```go
type Table struct {
    proc.Process[int]
    slots []int
}

func (t *Table) Init() error {
    if t.Args() < 1 { return fmt.Errorf("slot count must be positive") }
    t.slots = make([]int, t.Args())
    t.slots[0] = 42
    return nil
}

func (*Table) Run(ctx context.Context) { <-ctx.Done() }
func (t *Table) First() int { return t.slots[0] }

table, err := proc.Start[Table](root, 3)
if err != nil { return err }
value := table.Proc().First() // 42, even before Run starts.
```

Start sets the arguments and calls Init before callers can reach the object.
If Init returns an error, Start returns that error and no usable PID. If Init
panics, the panic reaches the caller. A process without its own Init uses the
empty implementation in `Process`. Only the runtime calls Init and Run.
The [complete example](../tests/testdata/accepted/processinit/main.go) also
shows a failed start and a retry with a reserved PID.

Use Init to create a channel too: the
[waker example](../tests/testdata/accepted/process/waker.go) needs no setup flag
or setup lock. Run and Wake receive the initialized channel.
If background work must finish later, expose that separate state through a
method such as `Ready() bool`.

For two processes that need each other's PID, Reserve one PID, start the first,
then call StartReserved for the reserved PID. Check its returned error.
Proc panics until initialization succeeds. A failed StartReserved leaves the
PID available for another attempt. Init must not call an unstarted peer.

## Restart and failure

If Run returns, it is not restarted, and its supervisor removes it. If Run
panics, the supervisor waits before restarting it: 1 second after the first
panic, then twice as long for each restart in the last 5 minutes. It creates a new struct from the same start
arguments, calls Init, and replaces the value behind the PID only when Init succeeds.
An Init error or panic counts as a failed attempt under the same restart limit.

During the delay and initialization, PID calls still use the old struct. A call already in progress
finishes on that old struct after the replacement. A method panic propagates
in the caller's goroutine. The supervisor handles it only if it leaves a
supervised Run. The restart rules apply to the process whose Run failed.

A sixth failure from the same child in 5 minutes stops its siblings, last started
first, and passes the failure to the parent supervisor. The parent restarts the
subtree with the same retry rule. If the failure reaches the root, the program
panics with the last panic value.

Pass a reporting function to `proc.NewRoot`, or pass `nil` to disable reports.
The function receives the value and stack of each recovered panic before the
restart delay. Nested supervisors use the same function. An escalation from a
nested supervisor is reported as a panic of its parent.
The function can run concurrently for different children. The supervisor waits
for it to return; a panic in the function is not recovered. Do not wait for
the same supervisor to stop from inside the function.

```go
root := proc.NewRoot(func(value any, stack []byte) {
    log.Printf("process panicked: %v\n%s", value, stack)
})
```

## Stop

`root.Stop()` returns a channel. Wait on it to wait for the supervised work to
stop. Children stop in reverse start order. Run must return when its context
is cancelled; Stop cannot force it to exit. Stop does not wait for direct method
calls. Init has no cancellation context and must return for a restarting process
to stop. A PID can still call methods on its last struct after Run ends or Stop
completes.

Use `proc.StartSupervisor(parent)` to stop a group together and restart that
group after a child exceeds the restart limit. In tests,
[proctest.NewRoot(t)](../pkg/proc/proctest/readme.md) stops the tree during
cleanup and fails if it does not stop within 5 seconds.

## Start children

Use `p.Supervisor()` when a process decides which workers to run. For example,
a dispatcher can add one group for each configuration entry and remove a group
when that entry is deleted. These children stop with the process. A restart
starts a new process; Init or Run builds its groups again. Children created in
Init wait until Run starts. If initialization fails, the runtime stops those
children. Init must close any files or other resources it opened before returning
an error; the runtime does not close application resources.

```go
proc.Start[Worker](p.Supervisor(), WorkerArgs{ID: 1})
```

To stop a group separately, start it with
`sub := proc.StartSupervisor(p.Supervisor())`, then wait on `<-sub.Stop()`.
The [child-process guide](process-children.md) shows a worker pool and a
dispatcher that changes its groups while it runs.

## Not checked

These checks do not isolate ordinary mutable state. Two processes can still
access the same global map or share data through ordinary helper functions.
Use locks for that data, and run tests with `go test -race`. The race detector
can find a race only when the test executes the conflicting accesses.

The checker does not check whether a child PID belongs to an old run. See
[child lifetimes](process-children.md#child-lifetimes) before storing or sharing
child PIDs.

## Reference

The [process reference](process-forms.md) lists every supported declaration and
operation, followed by rejected examples. The [API reference](../pkg/proc/readme.md)
contains function signatures. See [performance](performance.md) for operation costs.
