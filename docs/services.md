# Supervised services

Use `supervise` for state with background work, such as a cache that refreshes a file.

| API | Purpose |
| --- | --- |
| `Service[A]` | Mutable service state with start arguments of type A. |
| `Group` | Start, restart, and stop related services. |
| `Ref[T]` | Reach the current service after a restart replaces its struct. |

## Refresh a file and serve its contents

This cache reads a file before startup succeeds. It then refreshes the contents
at the configured interval. Calls to `Text` return the last contents without reading the
file again.

The [configuration project](../examples/configuration/README.md) runs this
[FileCache](../examples/configuration/filecache.go), a JSON configuration loader,
and a request limiter. Run it from the repository root:

```sh
go run ./examples/configuration/cmd/limits
```

The command reads [limits.json](../examples/configuration/limits.json). Edit a
limit while it runs and observe the changed request decisions. Press Ctrl+C to
stop the group. Its file cache starts with these arguments:

```go
cache, err := supervise.Start[configuration.FileCache](root, configuration.FileArgs{
    Path: path, Every: 100 * time.Millisecond,
})
if err != nil { panic(err) }
fmt.Print(cache.Current().Text())
```

The [command](../examples/configuration/cmd/limits/main.go) supplies `root` and
`path`. FileCache loads in Init, polls in Run, and protects its text with a mutex.

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
| Start arguments and results of methods called through Current | [Allowed stored types](common-forms.md#markers-and-stored-types), such as scalars, marked structs, frozen collections, and Refs. |
| Method inputs | Ordinary Go types, including interfaces, callbacks, pointers, slices, maps, and channels. Protected values follow the [common source rules](common-forms.md). |
| Return a list | `im.FrozenSlice[string]`. |
| Replace immutable configuration | An [im.Owner[Config]](owner.md) field. |

A reference can be a field of another service's arguments. The
[limiter](../examples/configuration/limiter.go) takes a reference to a
[configuration loader](../examples/configuration/config.go).
With those types, start both in one group:

```go
group := supervise.StartGroup(root)
loader, err := supervise.Start[configuration.ConfigLoader](group, configuration.LoaderArgs{
    Cache: cache, Every: 100 * time.Millisecond,
})
if err != nil { panic(err) }
limiter, err := supervise.Start[configuration.Limiter](group, configuration.LimiterArgs{
    Loader: loader, Key: "api", Window: time.Second,
})
if err != nil { panic(err) }
decision := limiter.Current().Take()
```

The loader parses the initial limits in Init. The limiter reads the current
limit through its reference and resets its request count every Window.
Wait on `<-group.Stop()` to stop both services. Use [child groups](service-children.md) when a service must
create or remove related work while it runs.

## Review one service

The API rules aim to keep review local. Review these together inside the service:

- Field access from Init, Run, and methods; the locks that protect it.
- How mutable inputs are used, copied, or saved.
- Whether callbacks or dependencies receive access to mutable service state.
- Child startup and shutdown.

Checked callers use methods through Ref; they cannot access service fields.
Results use allowed stored types. Method logic and ordinary input contracts
remain part of the service review.

## Startup errors and reservations

- Only the runtime calls Init and Run. Use the default Init when no setup is needed.
- A first Init error makes Start return an empty Ref and that error.
- A first Init panic reaches the Start caller.
- To pass a reference before startup: `ref := supervise.Reserve[T]()`, then
  `err := supervise.StartReserved(parent, ref, args)`. Failed Init permits a retry.
- A zero `Ref[T]` is empty, not reserved. Current panics on empty or unstarted Refs.
- Init must not call an unstarted peer. For readiness that depends on Run,
  expose a method such as `Ready() bool`.

The [startup and retry example](../examples/configuration/README.md#startup-and-retry) shows errors and a retry.

## Concurrent dependencies

Some start arguments are mutable values that several goroutines can use
safely, such as a logger or an HTTP client. Declare a named
`supervise.ConcurrentSafe[T]` field and wrap its value with
`supervise.AssumeConcurrentSafe`. The [model service](../examples/models/fetcher.go)
uses these arguments; [Store](../examples/models/store.go) saves each valid document:

```go
type FetcherArgs struct {
    im.Shared
    URL      string
    Every    time.Duration
    Log      supervise.ConcurrentSafe[*slog.Logger]
    Client   supervise.ConcurrentSafe[*http.Client]
    NewStore supervise.ConcurrentSafe[func(name string) Store]
}
```

Build the arguments where you start the service:

```go
args := models.FetcherArgs{
    URL:      "http://localhost:8080/model.json",
    Every:    time.Second,
    Log:      supervise.AssumeConcurrentSafe(logger),
    Client:   supervise.AssumeConcurrentSafe(http.DefaultClient),
    NewStore: supervise.AssumeConcurrentSafe(factory),
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
`supervise.ConcurrentSafe` is not an immutable stored type.
The [model project](../examples/models/README.md) downloads real model data,
saves it through the factory's store, and serves predictions during reloads.
Its [command](../examples/models/cmd/models/main.go) supplies `logger` and
`factory`; its tests use a local HTTP server and a shared recording store.

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
