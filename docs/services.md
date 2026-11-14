# Supervised services

Use `supervise` for state with background work, such as a cache that refreshes a file.

| API | Purpose |
| --- | --- |
| `Service[A]` | Mutable service state with start arguments of type A. |
| `Group` | Start, restart, and stop related services. |
| `Ref[T]` | Reach the current service after a restart replaces its struct. |

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

1. `Start[FileCache]` creates the cache, sets its arguments, and calls `Init`.
2. After Init succeeds, Start returns `Ref[FileCache]` and schedules Run in a
   goroutine. Run might not have started when Start returns.
3. `cache.Current().Text()` runs in the caller's goroutine. The mutex protects
   concurrent reads and refreshes. Keep `cache`; call methods directly on Current.
4. A later file-read error makes Run panic. The group waits, then creates a
   replacement with the same arguments. Calls use the old cache until the new
   Init succeeds; calls already in progress finish on the old cache.

## Choose arguments and results

| Need | Use |
| --- | --- |
| Start arguments | Embed `supervise.Service[A]`; read `p.Args()` in a service method. |
| No arguments | Embed `supervise.Service[supervise.NoArgs]`; pass `supervise.NoArgs{}` to Start. |
| Arguments or method inputs/results | [Allowed stored types](common-forms.md#markers-and-stored-types), such as scalars, marked structs, frozen collections, and Refs. |
| Return a list | `im.FrozenSlice[string]`. |
| Replace immutable configuration | An [im.Owner[Config]](owner.md) field. |

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

- Only the runtime calls Init and Run. Use the default Init when no setup is needed.
- A first Init error makes Start return an empty Ref and that error.
- A first Init panic reaches the Start caller.
- To pass a reference before startup: `ref := supervise.Reserve[T]()`, then
  `err := supervise.StartReserved(parent, ref, args)`. Failed Init permits a retry.
- A zero `Ref[T]` is empty, not reserved. Current panics on empty or unstarted Refs.
- Init must not call an unstarted peer. For readiness that depends on Run,
  expose a method such as `Ready() bool`.

The [startup example](../tests/testdata/accepted/serviceinit/main.go) shows errors and a retry.

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

| Operation | Example |
| --- | --- |
| Give the service these arguments | Embed `supervise.Service[FetcherArgs]`; pass `args` to Start. |
| Read a dependency inside a service method | `log := f.Args().Log.Value()` |
| Use it directly | `f.Args().Log.Value().Info("start")` |

`AssumeConcurrentSafe` is a caller promise, not a proof, lock, or freeze.
An incorrect promise can cause a data race. Use only the listed
[dependency forms](service-forms.md#concurrent-dependencies).
The [complete example](../tests/testdata/accepted/service/dependencies.go)
also shows a factory function and a test fake.

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

Report rules:

- Called before the delay for each recovered panic or restart Init error.
- Shared by nested groups; a failure passed to a parent is reported again.
- Can run concurrently for different children.
- Must return; the group waits for it and does not recover its panics.
- Must not wait for its own group to stop; that would wait for the report itself.

Shutdown rules:

- `Stop()` requests shutdown; `<-root.Stop()` waits for the tree to stop.
- Run must return when its context ends.
- Init has no cancellation context and must return before its attempt can finish.
- Stop does not wait for direct method calls.
- Current still returns the last published value after Run or Stop ends.

Create groups with NewRoot or StartGroup. A zero Group cannot run. Starting
under a stopped group still calls Init and can succeed, but Run will not run.

In tests, use
[`supervisetest.NewRoot(t)`](../pkg/supervise/supervisetest/readme.md). It stops
the tree during cleanup and fails the test if shutdown takes more than five
seconds.

## What the checker covers

| Checked by immutago | Still ordinary Go |
| --- | --- |
| Listed argument, method, Ref, and child-group forms | Mutable service fields, global maps, and data shared through ordinary helpers |

Synchronize shared mutable state. Use `go test -race` to check the accesses your tests execute.

Use the [service reference](service-forms.md) for all allowed source forms and
concrete rejected examples. The [API reference](../pkg/supervise/readme.md)
lists signatures and the [cost reference](performance.md) lists operation costs.
