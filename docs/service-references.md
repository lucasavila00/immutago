# Service references

`supervise.Ref[T]` identifies one service. Copy, store, or return the reference.
Call an exported service method with `ref.Current().Method(args)`.
Copies select the same service, including after a restart.

## Example: call a selected pool

The [worker project](../examples/workers/README.md) returns a pool reference inside
a service result:

```go
route := dispatcher.Current().Lookup("blue")
if route.Available {
    result := route.Pool.Current().Process("Hello World")
    fmt.Println(result.Text)
}
```

[Lookup](../examples/workers/lookup.go) reads the pool index under a lock.
The caller can keep `route.Pool` and call the pool directly.

## Reference operations — exhaustive

`T` must be a concrete struct that embeds `supervise.Service[A]`.
The examples use `WorkerRef`, an alias of `supervise.Ref[Worker]`.

| Form | Example |
| --- | --- |
| Start, reserve, or start a reserved reference | `ref, err := supervise.Start[Worker](group, args)`, `ref := supervise.Reserve[Worker]()`, `err := supervise.StartReserved(group, ref, args)` |
| Declare, construct, or allocate an empty reference | `var ref WorkerRef`, `WorkerRef{}`, `new(WorkerRef)` |
| Alias the reference type | `type WorkerRef = supervise.Ref[Worker]` |
| Copy or assign references, including multiple bindings and tuple results | `next := ref`, `first, second := ref, ref`, `ref, ok := workers[name]` |
| Pass or return references with concrete types, including variadic arguments | `use(ref)`, `useAll(ref, next)`, `useAll(workers...)`, `return ref` |
| Store references in variables, struct fields, arrays, slices, maps, or channels | `var current WorkerRef`, `struct{ Worker WorkerRef }{ref}`, `[2]WorkerRef{ref}`, `[]WorkerRef{ref}`, `map[string]WorkerRef{"blue": ref}`, `make(chan WorkerRef)` |
| Store references in marked fields or frozen collections | `type Route struct { im.ByValue; Worker WorkerRef }`, `im.FrozenSlice[WorkerRef]`, `im.FrozenMap[string, WorkerRef]` |
| Read or update ordinary reference storage; take an address or dereference a pointer | `ref := route.Worker`, `workers[name] = ref`, `p := &ref`, `next := *p` |
| Index or slice native storage | `workers[0]`, `workers[1:]` |
| Compare references or use them in a switch | `ref == next`, `ref != next`, `switch ref { case next: use(ref) }` |
| Range over native storage or an iterator with concrete reference types | `for _, ref := range workers { use(ref) }`, `for ref := range iterator { use(ref) }` |
| Allocate or change native containers | `make([]WorkerRef, 2)`, `append(workers, ref)`, `copy(dst, src)`, `delete(workers, name)`, `clear(workers)` |
| Read native container sizes | `len(workers)`, `cap(workers)` |
| Send, receive, or close a concrete channel | `ch <- ref`, `ref := <-ch`, `close(ch)` |
| Copy, pass, or call callbacks with concrete reference parameters and results | `func use(ref WorkerRef, call func(WorkerRef)) { call(ref) }`, `next := produce; ref := next()` where produce has type `func() WorkerRef` |
| Copy, pass, or return an interface with concrete reference parameters or results; keep the same interface type | `type Producer interface { Worker() WorkerRef }`, `type Consumer interface { Use(WorkerRef) }`, `next := producer` |
| Call these interface methods | `producer.Worker().Current().Method()`, `consumer.Use(ref)` |
| Capture a reference in a closure | `call := func() { ref.Current().Method() }` |
| Call an exported service method directly, including in go or defer | `ref.Current().Method()`, `go ref.Current().Method()`, `defer ref.Current().Method()` |
| Add parentheses to a direct call | `((ref.Current)()).Method()` |

Fixed `im` and `supervise` generic APIs are allowed. User generic type arguments
cannot contain references, including in fields, interfaces, or callback types.
User generics can name a concrete reference type without using it as a type argument:

| Form | Example |
| --- | --- |
| Concrete parameter or result | `func keep[T any](ref WorkerRef, value T) WorkerRef { return ref }` |
| Concrete field | `type Box[T any] struct { Worker WorkerRef; Value T }`; use `Box[int]`. |
| Concrete alias | `type Alias[T any] = WorkerRef`; use `Alias[int]`. |

Marked objects, frozen collections, and other im arguments or results keep the
[im rules](common-forms.md). For example, copy a reference out of a marked object
before a closure, go, or defer uses it:

```go
pool := route.Pool
go pool.Current().Process("Hello")
```

## Current calls — exhaustive

| Form | Rule |
| --- | --- |
| `ref.Current().Method(args)` | Method is an exported service method, including a promoted method, other than Init, Run, or a method supplied by the runtime. It returns no result or one [service result](service-results.md). |

The checker rejects every other use of `Current`, despite its Go return type
of `*T`. Current panics for an empty or unstarted reference. Each call selects the
last successfully initialized service; a call in progress finishes on that instance.

## Rejected examples — non-exhaustive

| Rejected | Use instead |
| --- | --- |
| `worker := ref.Current()` | Keep `ref`; call `ref.Current().Method()` each time. |
| `ref.Current().Count`, `*ref.Current() = Worker{}` | Add an exported service method for the operation. |
| `current := ref.Current`, `method := ref.Current().Method` | Use a closure that captures the reference and makes the direct method call. |
| `supervise.Ref[Worker].Current(ref)` | `ref.Current().Method()` |
| `ref.Current().Init()`, `ref.Current().Run(ctx)`, `ref.Current().Args()` | Initialize through Start; expose needed data through an exported service method. |
| `func read(WorkerRef) int` without a body | Supply a Go body so its reference use can be checked. |
| `var value any = ref`, `identity(ref)` with generic identity | Use a concrete `supervise.Ref[Worker]` parameter. |
| `WorkerRef(other)`, `value.(WorkerRef)` | Copy a value that already has type `WorkerRef`. |
| `var value any = produce` where produce returns WorkerRef | Keep the callback's concrete function type. |
| `var value any = producer` | Keep the exact `Producer` interface type. |
| `wrapper.Current()` through an embedded Ref | Select the reference field: `wrapper.Ref.Current().Method()`. |

## What the reference rule does not check

- Service fields can change. A Ref field in an immutable object cannot change,
  but the selected service can.
- Calls run in the caller's goroutine. They can run at the same time as Run and
  other calls. The service must protect shared mutable fields.
- A service method can give a callback or global access to its mutable state.
  The reference rule does not prevent this.

The [checker table and callback example](arch/services.md#service-reference-boundary)
show these checks and their limits.
