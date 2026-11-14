# Supervised processes

Use `proc` for a long-running service that should restart after a panic in its
background work, such as a configuration loader or cache refresher.
A process is a mutable Go struct. Its `Run` method runs in a goroutine managed
by a supervisor. Keep a `PID` to call the process; the PID still works after a
restart replaces the struct.

Method calls are ordinary Go calls in the caller's goroutine. There is no
mailbox and calls are not serialized. Use a mutex when methods and `Run` access
the same mutable fields. The checker requires immutable start arguments and
exported method parameters/results.

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

## Start and readiness

Start publishes the new struct before Run begins. A method can therefore run
before Run initializes any fields. Put required initial data in the start
arguments, and make methods handle the zero values of other fields.

For two processes that need each other's PID, Reserve one PID, start the first
process, then call StartReserved for the reserved PID. Reserve does not wait
for startup. Calling Proc before StartReserved panics; coordinate the first
calls so that both processes have started.

## Restart and failure

If Run returns, it is not restarted. If Run panics, the supervisor waits before
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

A supervisor recovers these panics, so nothing logs them by default. To see
them, pass `proc.OnPanic` to `NewRoot`. Its function gets the value and the
stack of each panic that a supervisor of the tree recovers, before the restart
delay. A nested supervisor that escalates is such a panic of its parent.
The function can run concurrently for different children. The supervisor waits
for it to return; a panic in the function is not recovered. Do not wait for
the same supervisor to stop from inside the function.

```go
root := proc.NewRoot(proc.OnPanic(func(value any, stack []byte) {
    log.Printf("process panicked: %v\n%s", value, stack)
}))
```

## Stop

`root.Stop()` returns a channel. Wait on it to wait for the supervised work to
stop. Children stop in reverse start order. Run must return when its context
is cancelled; Stop cannot force it to exit. Stop does not wait for direct method
calls. A PID can still call methods on its last struct after Run ends or Stop
completes.

Use `proc.StartSupervisor(parent)` to stop a group together and restart that
group after a child exceeds the restart limit. In tests, [proctest.NewRoot(t)](../pkg/proc/proctest/readme.md) stops the
tree during cleanup and fails if it does not stop within 5 seconds.

## Not checked

These checks do not isolate ordinary mutable state. Two processes can still
access the same global map or share data through ordinary helper functions.
Use locks for that data, and run tests with `go test -race`. The race detector
can find a race only when the test executes the conflicting accesses.

## Reference

The [process reference](process-forms.md) lists every supported declaration and
operation, followed by rejected examples. The [API reference](../pkg/proc/readme.md)
contains function signatures. See [performance](performance.md) for operation costs.
