# Service handles

`supervise.Ref[T]` identifies one service. Copy, store, or return the handle.
Call an exported message with `ref.Current().Message(args)`.
Copies select the same service, including after a restart.

## Example: call a selected pool

The [worker project](../examples/workers/README.md) returns a pool handle inside
a Scoped result:

```go
route := dispatcher.Current().Lookup("blue")
if route.Available {
    result := route.Pool.Current().Process("Hello World")
    fmt.Println(result.Text)
}
```

[Lookup](../examples/workers/lookup.go) reads the pool index under a lock.
The caller can keep `route.Pool` and call the pool directly.

## Handle operations — exhaustive

The tables in this section list all allowed handle forms. Unlisted handle
operations are rejected. Go type rules also apply.

`T` must be a concrete struct that embeds `supervise.Service[A]`.
The examples use a service named `Worker` and this alias:

```go
type WorkerRef = supervise.Ref[Worker]
```

| Form | Example |
| --- | --- |
| Start, reserve, or start a reserved handle | `ref, err := supervise.Start[Worker](group, args)`, `ref := supervise.Reserve[Worker]()`, `err := supervise.StartReserved(group, ref, args)` |
| Declare, construct, or allocate an empty handle | `var ref WorkerRef`, `WorkerRef{}`, `new(WorkerRef)` |
| Alias the handle type | `type WorkerRef = supervise.Ref[Worker]` |
| Copy or assign handles, including multiple bindings and tuple results | `next := ref`, `first, second := ref, ref`, `ref, ok := workers[name]` |
| Pass or return handles with concrete types, including variadic arguments | `use(ref)`, `useAll(ref, next)`, `useAll(workers...)`, `return ref` |
| Store handles in variables, struct fields, arrays, slices, maps, or channels | `var current WorkerRef`, `struct{ Worker WorkerRef }{ref}`, `[2]WorkerRef{ref}`, `[]WorkerRef{ref}`, `map[string]WorkerRef{"blue": ref}`, `make(chan WorkerRef)` |
| Store handles in marked fields or frozen collections | `type Route struct { im.Scoped; Worker WorkerRef }`, `im.FrozenSlice[WorkerRef]`, `im.FrozenMap[string, WorkerRef]` |
| Read or update ordinary handle storage; take an address or dereference a pointer | `ref := route.Worker`, `workers[name] = ref`, `p := &ref`, `next := *p` |
| Index or slice native storage | `workers[0]`, `workers[1:]` |
| Compare handles or use them in a switch | `ref == next`, `ref != next`, `switch ref { case next: use(ref) }` |
| Range over native storage or an iterator with concrete handle types | `for _, ref := range workers { use(ref) }`, `for ref := range iterator { use(ref) }` |
| Allocate or change native containers | `make([]WorkerRef, 2)`, `append(workers, ref)`, `copy(dst, src)`, `delete(workers, name)`, `clear(workers)` |
| Read native container sizes | `len(workers)`, `cap(workers)` |
| Send, receive, or close a concrete channel | `ch <- ref`, `ref := <-ch`, `close(ch)` |
| Copy, pass, or call callbacks with concrete handle parameters and results | `func use(ref WorkerRef, call func(WorkerRef)) { call(ref) }`, `next := produce; ref := next()` where produce has type `func() WorkerRef` |
| Copy, pass, return, or call a producer interface with the same interface type | `type Producer interface { Worker() WorkerRef }`, `next := producer`, `producer.Worker().Current().Message()` |
| Capture a handle in a closure | `call := func() { ref.Current().Message() }` |
| Call a message directly, including in go or defer | `ref.Current().Message()`, `go ref.Current().Message()`, `defer ref.Current().Message()` |
| Add parentheses to a direct call | `((ref.Current)()).Message()` |

Fixed `im` and `supervise` generic APIs are allowed. User generic type arguments
cannot contain handles, including in fields, interfaces, or callback types.
User generics can name a concrete handle type without using it as a type argument:

| Form | Example |
| --- | --- |
| Concrete parameter or result | `func keep[T any](ref WorkerRef, value T) WorkerRef { return ref }` |
| Concrete field | `type Box[T any] struct { Worker WorkerRef; Value T }`; use `Box[int]`. |
| Concrete alias | `type Alias[T any] = WorkerRef`; use `Alias[int]`. |

Marked objects, frozen collections, and other im arguments or results keep the
[im rules](common-forms.md). For example, copy a handle out of a marked object
before a closure, go, or defer uses it:

```go
ref := route.Worker
go ref.Current().Message()
```

## Current calls — exhaustive

| Form | Rule |
| --- | --- |
| `ref.Current().Message(args)` | Message is an exported user method, including a promoted method, other than Init or Run. It returns nothing or one [Scoped result](scoped-results.md). |

The checker rejects every other use of `Current`, despite its Go return type
of `*T`. Current panics for an empty or unstarted handle. Each call selects the
last initialized service; a call in progress finishes on that instance.

## Rejected examples — non-exhaustive

| Rejected | Use instead |
| --- | --- |
| `worker := ref.Current()` | Keep `ref`; call `ref.Current().Message()` each time. |
| `ref.Current().Count`, `*ref.Current() = Worker{}` | Add a message method for the operation. |
| `current := ref.Current`, `method := ref.Current().Message` | Use a closure that captures the handle and makes the direct message call. |
| `supervise.Ref[Worker].Current(ref)` | `ref.Current().Message()` |
| `ref.Current().Init()`, `ref.Current().Run(ctx)`, `ref.Current().Args()` | Initialize through Start; expose needed data through a message. |
| `func read(WorkerRef) int` without a body | Supply a Go body so its handle use can be checked. |
| `var value any = ref`, `identity(ref)` with generic identity | Use a concrete `supervise.Ref[Worker]` parameter. |
| `WorkerRef(other)`, `value.(WorkerRef)` | Copy a value that already has type `WorkerRef`. |
| `var value any = produce` where produce returns WorkerRef | Keep the callback's concrete function type. |
| `var value any = producer` | Keep the exact `Producer` interface type. |
| `wrapper.Current()` through an embedded Ref | Select the handle field: `wrapper.Ref.Current().Message()`. |

## What the handle rule does not check

- Service fields can change. A Ref field in an immutable object cannot change,
  but the selected service can.
- Calls run in the caller's goroutine. They can run at the same time as Run and
  other calls. The service must protect shared mutable fields.
- A service method can give a callback or global access to its mutable state.
  The handle rule does not prevent this.

The [checker table and callback example](arch/ownership.md#service-handle-boundary)
show these checks and their limits.
