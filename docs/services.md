# Supervised services

Use `supervise` to start, restart, and stop background services.
A **service** is a struct that embeds `supervise.Service[A]`. The runtime calls
its `Init` and `Run` methods. Other service method calls run in the caller's
goroutine; the runtime does not queue them.

| API | Purpose |
| --- | --- |
| `Service[A]` | Supply start arguments and child groups to a service struct. |
| `Group` | Start, restart, and stop related services. |
| `Ref[T]` | Keep a service reference and select its current value after a restart. |

## Service contract

| Item | Rule |
| --- | --- |
| Init and Run | `*T` must provide `Init() error` and `Run(context.Context)`. Embedded `Service[A]` supplies a default Init and the runtime methods. |
| Start arguments | Read them with `s.Args()`. The runtime saves them for each restart. Ordinary Go types are allowed; protected data must satisfy the [stored-type rules](common-forms.md#markers-and-stored-types). |
| No arguments | Embed `supervise.Service[supervise.NoArgs]`; pass `supervise.NoArgs{}` to Start. |
| Exported service method result | Return no result or one [service result](service-results.md). This includes promoted methods. Init, Run, and methods supplied by the runtime are exempt. |
| Fields, method parameters, closures, and goroutines | Ordinary Go rules. Synchronize shared mutable data. Uses of protected values follow the [im rules](common-forms.md). |
| Service construction and groups | Ordinary Go rules. |
| Service references | Copy, store, and return `Ref[T]`. Call through `ref.Borrow().Method(args)`; direct Ref calls cannot save the pointer. Interface calls can expose it. See [service references](service-references.md). |

Review these together in the service code:

- Field access from Init, Run, and methods; the locks that protect it.
- How mutable inputs are used, copied, or saved.
- Whether callbacks or dependencies receive access to mutable service state.
- Child startup and shutdown.

Service methods can give callbacks or globals access to mutable state; see the
[callback example](arch/services.md#service-reference-boundary). Review the Go
contracts of callers and dependencies. Use `go test -race` for the accesses your
tests run.

## Example: delivery quotes

The [shipping application](../examples/shipping/README.md) quotes delivery prices
from a catalog file. One Loader service reads the first catalog in Init and
checks for edits in Run. Invalid edits leave the last valid catalog available.

```sh
go run ./examples/shipping/cmd/shipping
```

Start its loader with the file path and reload interval:

```go
loader, err := supervise.Start[shipping.Loader](root, shipping.Args{
    Path: "examples/shipping/catalog.json", Every: time.Second,
})
if err != nil { panic(err) }
```

For each request, the HTTP handler selects one catalog and calculates all parcel
prices from it:

```go
catalog := loader.Borrow().Catalog()
response := shipping.Quote(catalog, request)
```

`Catalog` runs in the caller's goroutine. It loads one generated projection from
the Cell. A reload can publish a new catalog while this quote finishes with the
selected version. See the [loader](../examples/shipping/loader.go) and
[quote calculation](../examples/shipping/quote.go).

## Startup errors and reservations

- Start and restarts call Init and Run. Use the default Init when no setup is needed.
- A first Init error makes Start return an empty Ref and that error.
- A first Init panic reaches the Start caller.
- To pass a reference before startup: `ref := supervise.Reserve[T]()`, then
  `err := supervise.StartReserved(parent, ref, args)`. Failed Init permits a retry.
- A zero `Ref[T]` is empty, not reserved. Borrow panics on empty or unstarted references.
- Init must not call an unstarted peer. For readiness that depends on Run,
  expose a method such as `Ready() bool`.

## Example: mutable dependencies

Start arguments can contain ordinary Go interfaces, clients, loggers, and
functions. The [search indexer](../examples/search/indexer.go) receives an fs.FS:

```go
indexer, err := supervise.Start[search.Indexer](root, search.IndexerArgs{
    Files: os.DirFS("examples/search/documents"), Every: time.Second,
})
if err != nil { panic(err) }
handler := search.HTTPHandler(indexer)
```

The [search application](../examples/search/README.md) reads files through that
interface. Its tests supply controlled filesystems to check edits and failed
scans. Review shared mutable dependencies under their normal Go contracts.

## Failure and shutdown

| Event | What happens |
| --- | --- |
| Run returns normally | The service ends and its group removes it. |
| Run panics | The group waits, creates a new struct, calls Init, and starts Run again after Init succeeds. |
| Init fails during a restart | The attempt fails. The reference continues to use the previous struct. |
| A direct service method call panics | The panic stays in its caller's goroutine. A call from Run or a restart Init can fail that attempt. |
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
- Nested groups use the same report function; a failure passed to a parent is reported again.
- Can run concurrently for different children.
- Must return; the group waits for it and does not recover its panics.
- Must not wait for its own group to stop; that would wait for the report itself.

Shutdown rules:

- `Stop()` requests shutdown; `<-root.Stop()` waits for the tree to stop.
- Run must return when its context ends.
- Init has no cancellation context and must return before its attempt can finish.
- Stop does not wait for direct method calls.
- Borrow still returns the last published value after Run or Stop ends.

Create groups with NewRoot or StartGroup. A zero Group cannot run. Starting
under a stopped group still calls Init and can succeed, but Run will not run.

In tests, use
[`supervisetest.NewRoot(t)`](../pkg/supervise/supervisetest/readme.md). It stops
the tree during cleanup and fails the test if shutdown takes more than five
seconds.
