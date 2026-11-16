# Service references and pointers

| Type | Use |
| --- | --- |
| `supervise.Ref[T]` | Store the service reference. Borrow selects its latest initialized instance. |
| `*T` outside the package that declares T | Keep one instance in a local variable; call its exported service methods. |
| `*T` inside the package that declares T | Ordinary Go rules for service implementation code. |

```go
worker := ref.Borrow()
limit := worker.Limit(id)
used := worker.Used(id)
```

Both calls use the same instance, even if the service restarts between them.
The methods still need to synchronize mutable state.

## Reference forms — exhaustive

T must be a concrete struct with a [value embedding path](services.md#service-embedding--exhaustive)
to `supervise.Service[A]`. Ref values use ordinary Go storage and call forms:

| Form | Example |
| --- | --- |
| Copy, store, or return | `other := ref`, `type Server struct { Worker supervise.Ref[Worker] }`, `return ref` |
| Use pointers, containers, or atomic storage | `p := &ref`, `[]supervise.Ref[Worker]{ref}`, `slot.Store(&ref)` |
| Convert a reference type | `type Handle supervise.Ref[api.Worker]`, `ref = supervise.Ref[api.Worker](handle)` |
| Use interfaces, generics, callbacks, go, or defer | `var value any = ref`, `identity(ref)`, `func() { use(ref) }`, `go use(ref)` |
| Call or select Borrow | `ref.Borrow()`, `next := ref.Borrow`, `supervise.Ref[Worker].Borrow(ref)` |
| Call through an interface, type parameter, or embedded Ref | `selected.Borrow()`, `wrapper.Borrow()` |
| Test an optional reference | `ref.IsZero()` |

A Ref field inside a Frozen or Local record keeps that record's
[common forms](common-forms.md). For example, copy a Ref field out of a Local
record before capturing it in a closure. A Frozen record's Ref field cannot be
replaced. Select an embedded Ref field explicitly in a marked record.

## Local service pointers — exhaustive

These rules depend on the service type's declaring package, not the spelling of
Borrow. Aliases keep the original package. Pointers to anonymous service structs
are Local in every package. Ordinary service values keep Go rules.

The examples use `api.Worker`, declared in another package.

| Pointer source | Example |
| --- | --- |
| A Ref call, including indirect calls | `worker := ref.Borrow()`, `worker := selected.Borrow()`, `worker := next()` |
| A factory or variable from the declaring package | `worker := api.NewWorker()`, `worker := api.Current` |
| A direct typed parameter | `func read(worker *api.Worker) int { return worker.Count() }` |
| Ordinary construction or a local address | `worker := new(api.Worker)`, `worker := &api.Worker{}`, `worker := &value` |
| A zero declaration or nil | `var worker *api.Worker`, `worker = nil` |

A function or interface that produces service pointers can use ordinary Go call
forms. The resulting pointers use this table. Contracts with Local-pointer
inputs or im.Local outputs still use the [common call rules](common-forms.md).

| Pointer use | Example |
| --- | --- |
| Bind, copy, replace, or discard locally | `other := worker`, `worker = ref.Borrow()`, `_ = worker` |
| Compare with a pointer or nil | `worker == other`, `worker != nil` |
| Pass to a direct call with an exact pointer parameter | `read(worker)`, `read(ref.Borrow())` |
| Call an exported service method, including a promoted method | `worker.Count()`, `ref.Borrow().Count()`; exclude Init, Run, and runtime methods. |
| Use pointer aliases | `type Pointer = *api.Worker` |

Combine these with the [common statement forms](common-forms.md#statements-and-calls).
Method results use the [service result rules](service-results.md).
Unlisted Local pointer operations are rejected.

## Rejected examples — non-exhaustive

These examples apply outside the service's declaring package.

| Rejected | Use instead |
| --- | --- |
| `var saved *api.Worker` or `type State struct { Worker *api.Worker }` | Store `supervise.Ref[api.Worker]`; borrow inside the function that needs it. |
| `return worker` | Return the Ref, or return the data read from a method. |
| `worker.CountField`, `*worker`, `worker.CountField = 1` | Call an exported service method. |
| `method := worker.Count` | Call `worker.Count()` directly. |
| `worker.Init()`, `worker.Run(ctx)`, `worker.Args()`, `worker.Children()` | Use Start and the exported service methods. |
| `func() int { return worker.Count() }` | Capture the Ref; borrow inside the closure. |
| `go read(worker)`, `defer read(worker)` | Call `read(worker)` directly. |
| `any(worker)`, `identity(worker)`, `[]*api.Worker{worker}` | Pass or store the Ref. |
| `&worker` or a protected pointer conversion | Use the existing local pointer with its declared type. |
| Pass worker as a service start argument | Pass its Ref. |

## Runtime behavior

| Operation | Behavior |
| --- | --- |
| Borrow | One atomic load. Shared access; no release call. |
| Retry publishes a new instance | Later Borrow calls select it. Existing pointers still select their original instance. |
| Method call | Runs in the caller's goroutine; can overlap Run and other calls. |
| Stop | Does not wait for method calls or invalidate the last instance. |
| Zero Ref | IsZero returns true. Borrow panics with `supervise: zero Ref`. |
| Successful Start | Returns a nonzero Ref. It stays nonzero after Stop; copies give the same IsZero result. |

Review mutable state, locks, and input use in the service package. The checker
does not prove service isolation or synchronization.
