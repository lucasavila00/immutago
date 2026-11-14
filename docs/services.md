# Supervised services

Use `supervise` for application state with background work, such as a cache
that refreshes a file. A **service** is a mutable Go struct. A **group** starts
services, restarts failed background work, and stops related services together.
A **reference** lets callers reach a service after a restart replaces its struct.

## Refresh a file and serve its contents

This cache reads a file before startup succeeds. It then refreshes the contents
once per second. Calls to `Text` return the last contents without reading the
file again.

Create `settings.txt` with the text `ready`, then run this
[complete example](../tests/testdata/accepted/servicedoc/main.go):

```go
package main

import (
    "context"
    "fmt"
    "os"
    "sync"
    "time"

    "github.com/lucasavila00/immutago/pkg/im"
    "github.com/lucasavila00/immutago/pkg/supervise"
)

// FileArgs selects a file and its refresh interval.
type FileArgs struct {
    im.Shared
    Path  string
    Every time.Duration
}

// FileCache serves the last file contents.
type FileCache struct {
    supervise.Service[FileArgs]
    mu   sync.RWMutex
    text string
}

// Init loads the first contents before calls begin.
func (c *FileCache) Init() error {
    if c.Args().Every <= 0 {
        return fmt.Errorf("refresh interval must be positive")
    }
    return c.read()
}

// Run refreshes the file until ctx ends.
func (c *FileCache) Run(ctx context.Context) {
    ticker := time.NewTicker(c.Args().Every)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := c.read(); err != nil {
                panic(err)
            }
        }
    }
}

// read replaces the contents after a successful read.
func (c *FileCache) read() error {
    path := c.Args().Path
    data, err := os.ReadFile(path)
    if err != nil {
        return err
    }
    c.mu.Lock()
    c.text = string(data)
    c.mu.Unlock()
    return nil
}

// Text returns the last contents.
func (c *FileCache) Text() string {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.text
}

// main starts the cache, reads it, and stops its group.
func main() {
    root := supervise.NewRoot(nil)
    cache, err := supervise.Start[FileCache](root, FileArgs{
        Path: "settings.txt", Every: time.Second,
    })
    if err != nil {
        panic(err)
    }
    fmt.Print(cache.Current().Text())
    <-root.Stop()
}
```

Check the program with `immutago ./...`, then run it with `go run .`. See
[Get started](getting-started.md) for installation. The example stops after
its first read; an application keeps the root group until shutdown.

`Start[FileCache]` creates a new cache, sets its arguments, and calls `Init`.
After `Init` succeeds, callers can read through the returned
`supervise.Ref[FileCache]`. `Run` starts in a managed goroutine. It need not have
started when `Start` returns.

`cache.Current().Text()` is an ordinary Go call in the caller's goroutine.
Several callers and `Run` can run at the same time. The cache uses a mutex to
protect the text. Keep the reference and call a method directly on `Current()`;
do not save the returned service pointer.

If a later file read fails, this example panics from `Run`. The group waits,
then builds a new cache with the same arguments. Calls keep using the old
cache until the new cache's `Init` succeeds. Each call already in progress
finishes on the old cache.

## Choose arguments and results

Embed `supervise.Service[A]` to declare the start argument type `A`. Read that
value with `Args()` inside a service method. For a service without arguments,
use `supervise.Service[supervise.NoArgs]` and pass `supervise.NoArgs{}` to Start.

Start arguments and the parameters/results of methods called through Current
use the [allowed stored types](common-forms.md#markers-and-stored-types). These include
scalars, marked structs, frozen collections, and service references. For
example, return a list as `im.FrozenSlice[string]`. Use an `im.Owner[Config]`
field when readers must use an immutable configuration while the service
publishes its next version; see [Owner](owner.md).

A reference can be a field of another service's arguments. The
[limiter](../tests/testdata/accepted/service/limiter.go) takes a reference to a
[configuration loader](../tests/testdata/accepted/service/config_loader.go).
With those types, start both in one group:

```go
group := supervise.StartGroup(root)
loader, err := supervise.Start[ConfigLoader](group, LoaderArgs{
    Path: "api", Every: time.Minute,
})
if err != nil { panic(err) }
limiter, err := supervise.Start[Limiter](group, LimiterArgs{
    Loader: loader, Key: "api",
})
if err != nil { panic(err) }
decision := limiter.Current().Take()
```

The limiter calls the loader through that reference. It returns `Limited`
until the loader publishes its first limit. Wait on `<-group.Stop()` to stop
both services. Use [child groups](service-children.md) when a service must
create or remove related work while it runs.

## Startup errors and reservations

If the first Init returns an error, Start returns that error and an empty
reference. An Init panic reaches the Start caller. Only the runtime calls Init
and Run. A service that needs no setup can use the default Init from Service.

To pass a reference before its service exists, create it with
`supervise.Reserve[T]()`, then start it with
`supervise.StartReserved(parent, ref, args)`. Check the returned error. A failed
Init leaves the reserved reference available for another attempt. The
[startup example](../tests/testdata/accepted/serviceinit/main.go) shows both
startup errors and a retry.

The zero value of `Ref[T]` is empty; it is not a reservation. `Current()` panics
until a reserved reference has completed initialization. Init must not call an
unstarted peer. If later background work determines readiness, expose that
separate state with a method such as `Ready() bool`.

## Concurrent dependencies

Some start arguments are mutable values that several goroutines can use
safely, such as a logger or an HTTP client. Declare a named
`supervise.ConcurrentSafe[T]` field and wrap its value with
`supervise.AssumeConcurrentSafe`:

```go
type FetcherArgs struct {
    im.Shared
    BaseURL string
    Log     supervise.ConcurrentSafe[*slog.Logger]
    Client  supervise.ConcurrentSafe[*http.Client]
}
```

Build the arguments where you start the service:

```go
args := FetcherArgs{
    BaseURL: "https://example.com",
    Log:     supervise.AssumeConcurrentSafe(logger),
    Client:  supervise.AssumeConcurrentSafe(http.DefaultClient),
}
```

Pass args to Start for a service that embeds `supervise.Service[FetcherArgs]`.
Inside its methods, read the logger with `f.Args().Log.Value()`. For example,
`f.Args().Log.Value().Info("start")` writes a log message. The value can also be
saved in an ordinary local, such as `log := f.Args().Log.Value()`. The complete
[dependency example](../tests/testdata/accepted/service/dependencies.go) also
passes a factory function and uses a test fake.

`AssumeConcurrentSafe` records the caller's promise. It does not freeze the
value, add synchronization, or prove that concurrent use is safe. A false
promise can cause a data race. The checker permits this wrapper only in the
listed [dependency forms](service-forms.md#concurrent-dependencies); it is not
an immutable stored type.

## Failure and shutdown

| Event | What happens |
| --- | --- |
| Run returns normally | The service ends and its group removes it. |
| Run panics | The group waits, creates a new struct, calls Init, and starts Run again after Init succeeds. |
| Init fails during a restart | The attempt fails. The reference continues to use the previous struct. |
| An ordinary method panics | The panic stays in its caller's goroutine. A call from Run or a restart Init can fail that attempt. |
| A child fails six times within five minutes | Its group stops all children, last started first, and passes the failure to its parent. |
| A failure reaches the root group | The program panics with the last failure value. |

The restart delay is one second after the first failure. It doubles for each
restart in the last five minutes. A nested group's parent restarts that group
with the same rule after a child exceeds the limit.

Pass a report function to `NewRoot`, or pass `nil` to disable reports:

```go
root := supervise.NewRoot(func(value any, stack []byte) {
    log.Printf("service failure: %v\n%s", value, stack)
})
```

The report receives each recovered panic or restart Init error before the
delay. Nested groups use the same report function. A failure passed up by a
nested group is reported again by its parent. Reports can run concurrently for
different children. The group waits for a report to return, and does not
recover a panic in that function. Do not wait for that same group to stop from
its report function.

`Stop()` requests shutdown and returns a completion channel. Wait on
`<-root.Stop()` to wait for the tree to stop. Each Run must observe its context
and return. Init has no cancellation context and must return before an attempt
can finish. Stop does not wait for direct method calls.

Construct groups with NewRoot or StartGroup; a zero Group is not ready to run.
Starting under a stopped group still calls Init and can return success, but Run
will not run. `Current()` means the last published value, not a running service:
calls can still use that value after Run ends or Stop completes.

In tests, use
[`supervisetest.NewRoot(t)`](../pkg/supervise/supervisetest/readme.md). It stops
the tree during cleanup and fails the test if shutdown takes more than five
seconds.

## What the checker covers

The checker checks the listed argument, method, reference, and child-group
forms. Ordinary mutable state still follows Go rules. For example, two services
can access a global map or share values through ordinary helpers. Synchronize
that state and test with `go test -race`; the race detector checks accesses that
the tests execute.

Use the [service reference](service-forms.md) for all allowed source forms and
concrete rejected examples. The [API reference](../pkg/supervise/readme.md)
lists signatures and the [cost reference](performance.md) lists operation costs.
