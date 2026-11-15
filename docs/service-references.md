# Service references

`supervise.Ref[T]` identifies one service. Copy, store, or return the reference.
Call an exported service method with `ref.Borrow().Method(args)`.
Copies select the same service, including after a restart.

## Example: select a catalog for a quote

The [shipping HTTP handler](../examples/shipping/http.go) keeps a reference to
its catalog loader. Each request selects the current loader, then one catalog:

```go
catalog := loader.Borrow().Catalog()
response := shipping.Quote(catalog, request)
```

Keep the loader reference for later requests. The selected catalog projection
stays local to the quote calculation.

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
| Store references in marked fields or frozen collections | `type Route struct { im.Frozen; Worker WorkerRef }`, `im.FrozenSlice[WorkerRef]`, `im.FrozenMap[string, WorkerRef]` |
| Read or update ordinary reference storage; take an address or dereference a pointer | `ref := route.Worker`, `workers[name] = ref`, `p := &ref`, `next := *p` |
| Index or slice native storage | `workers[0]`, `workers[1:]` |
| Compare references or use them in a switch | `ref == next`, `ref != next`, `switch ref { case next: use(ref) }` |
| Range over native storage or an iterator with concrete reference types | `for _, ref := range workers { use(ref) }`, `for ref := range iterator { use(ref) }` |
| Allocate or change native containers | `make([]WorkerRef, 2)`, `append(workers, ref)`, `copy(dst, src)`, `delete(workers, name)`, `clear(workers)` |
| Read native container sizes | `len(workers)`, `cap(workers)` |
| Send, receive, or close a concrete channel | `ch <- ref`, `ref := <-ch`, `close(ch)` |
| Copy, pass, or call callbacks with concrete reference parameters and results | `func use(ref WorkerRef, call func(WorkerRef)) { call(ref) }`, `next := produce; ref := next()` where produce has type `func() WorkerRef` |
| Copy, pass, or return an interface with concrete reference parameters or results | `type Producer interface { Worker() WorkerRef }`, `type Consumer interface { Use(WorkerRef) }`, `next := producer` |
| Convert a reference or any value that contains references to an interface, explicitly or implicitly | `var value any = ref`, `any(ref)`, `consume(ref)` where consume takes any, `return ref` from a function that returns any, `struct{ Value any }{ref}` |
| Convert holders, embedded references, native containers, or callbacks to interfaces | `any(struct{ Worker WorkerRef }{ref})`, `any(struct{ WorkerRef }{ref})`, `any([]WorkerRef{ref})`, `any(func() WorkerRef { return ref })` |
| Convert between interfaces, including different reference method contracts | `var value any = producer`, `Producer(value.(Producer))` |
| Recover a reference or a value that contains references from an interface | `ref := value.(WorkerRef)`, `ref, ok := value.(WorkerRef)`, `switch ref := value.(type) { case WorkerRef: use(ref) }` |
| Use interface destinations in native storage, comparisons, switches, and range bindings | `ch <- ref` where ch has type chan any, `map[any]any{ref: ref}`, `ref == value`, `switch value { case ref: use(ref) }`, `for _, value = range workers { use(value) }` where value has type any |
| Call these interface methods | `producer.Worker().Borrow().Method()`, `consumer.Use(ref)` |
| Capture a reference in a closure | `call := func() { ref.Borrow().Method() }` |
| Call an exported service method directly, including in go or defer | `ref.Borrow().Method()`, `go ref.Borrow().Method()`, `defer ref.Borrow().Method()` |
| Add parentheses to a direct call | `((ref.Borrow)()).Method()` |

### Atomic reference storage — exhaustive

Use `sync/atomic.Pointer[T]` when goroutines read and replace a selected reference.
T follows the reference storage rules above. It can be a reference or contain
references in struct fields, native containers, or callback signatures.
For example, `current` can be a field with type `atomic.Pointer[WorkerRef]`:

```go
current.Store(&ref)
if selected := current.Load(); selected != nil {
    count := selected.Borrow().Count()
    _ = count
}
```

| Form | Example |
| --- | --- |
| Declare, construct, or allocate the standard pointer type | `var current atomic.Pointer[WorkerRef]`, `atomic.Pointer[WorkerRef]{}`, `new(atomic.Pointer[WorkerRef])` |
| Store a type that contains references | `type Slot struct { Ref WorkerRef }`; `var current atomic.Pointer[Slot]`; `var produce atomic.Pointer[func() WorkerRef]` |
| Use an import alias, a concrete type alias, or a generic alias with one type parameter that forwards it unchanged | `import atom "sync/atomic"`; `type Selected = atom.Pointer[WorkerRef]`; `type Pointer[T any] = atom.Pointer[T]`; `var current Pointer[WorkerRef]` |
| Load, store, swap, or compare and swap typed pointers, including nil | `selected := current.Load()`, `current.Store(&ref)`, `previous := current.Swap(&ref)`, `changed := current.CompareAndSwap(previous, nil)` |

The reference operations above apply to these typed values. The atomic type does
not change the Borrow call rule or the im type and storage rules. Do not change
the pointed-to value while another goroutine reads it. Do not copy an atomic
pointer after its first use.

### Generic reference types — exhaustive

Fixed `im` and `supervise` generic APIs and the atomic forms above are allowed.
Other generic type arguments cannot contain references, including in fields,
interfaces, or callback types.
User generics can name a concrete reference type without using it as a type argument:

| Form | Example |
| --- | --- |
| Concrete parameter or result | `func keep[T any](ref WorkerRef, value T) WorkerRef { return ref }` |
| Concrete field | `type Box[T any] struct { Worker WorkerRef; Value T }`; use `Box[int]`. |
| Concrete alias | `type Alias[T any] = WorkerRef`; use `Alias[int]`. |

Marked objects, frozen collections, and other im arguments or results keep the
[im rules](common-forms.md). Copy a reference out of a marked object before a
closure, go, or defer uses it.

## Borrow calls — exhaustive

| Form | Rule |
| --- | --- |
| `ref.Borrow().Method(args)` with a concrete Ref receiver | Method is an exported service method, including a promoted method, other than Init, Run, or a method supplied by the runtime. It returns no result or one [service result](service-results.md). |
| `selected.Borrow()` with an interface receiver, including method values and expressions | Ordinary Go rules apply: `pointer := selected.Borrow()`, `accessor := selected.Borrow`, `Borrower.Borrow(selected)` where Borrower is the interface type. The result can be stored and used outside the checked call form. |

For direct Ref calls, the checker rejects every other use of `Borrow`, despite
its Go return type of `*T`. Borrow panics for an empty or unstarted reference.
Each call selects the last successfully initialized service; a call in progress
finishes on that instance.
Borrow is shared and is not exclusive. No release call is needed.

Interface conversion can expose the service pointer outside this checked call
form. This is accepted, including when an embedded Ref promotes Borrow:

```go
var selected interface{ Borrow() *Worker } = ref
pointer := selected.Borrow()
```

The direct Ref call rule does not apply to this interface call. Callers must
account for restarts when they save a service pointer.

## Rejected examples — non-exhaustive

| Rejected | Use instead |
| --- | --- |
| `worker := ref.Borrow()` | Keep `ref`; call `ref.Borrow().Method()` each time. |
| `ref.Borrow().Count`, `*ref.Borrow() = Worker{}` | Add an exported service method for the operation. |
| `current := ref.Borrow`, `method := ref.Borrow().Method` | Use a closure that captures the reference and makes the direct method call. |
| `supervise.Ref[Worker].Borrow(ref)` | `ref.Borrow().Method()` |
| `ref.Borrow().Init()`, `ref.Borrow().Run(ctx)`, `ref.Borrow().Args()` | Initialize through Start; expose needed data through an exported service method. |
| `func read(WorkerRef) int` without a body | Supply a Go body so its reference use can be checked. |
| `identity(ref)` with generic identity | Use a concrete `supervise.Ref[Worker]` parameter, or pass `identity(any(ref))`. |
| `WorkerRef(other)` where other has a different defined concrete type | Use a reference type alias and copy the value. |
| `wrapper.Borrow()` through an embedded Ref | Select the reference field: `wrapper.Ref.Borrow().Method()`. |
| `type Box[T any] struct { Value T }`; `atomic.Pointer[Box[WorkerRef]]` | Use `type Slot struct { Ref WorkerRef }` and `atomic.Pointer[Slot]`. |
| `type Pointer[T any] = atomic.Pointer[Box[T]]`; `Pointer[WorkerRef]` | Use `type Pointer[T any] = atomic.Pointer[T]` and a concrete Slot type. |
| `type Pointer[T any] atomic.Pointer[T]`; `Pointer[WorkerRef]` | Use a type alias: `type Pointer[T any] = atomic.Pointer[T]`. |

## What the reference rule does not check

- Service fields can change. A Ref field in an immutable object cannot change,
  but the selected service can.
- Calls run in the caller's goroutine. They can run at the same time as Run and
  other calls. The service must protect shared mutable fields.
- A service method can give a callback or global access to its mutable state.
  The reference rule does not prevent this.
