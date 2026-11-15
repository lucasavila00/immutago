# Service references

`supervise.Ref[T]` identifies a service. Copy, store, or return the reference;
call an exported service method with `ref.Current().Method(args)`.
After a restart, Current selects the replacement service value.
A Ref field fixes which service the object names; it does not freeze that service's state.

## Example: call a selected pool

The [worker project](../examples/workers/README.md) returns a pool reference in
a ByValue struct:

```go
route := dispatcher.Current().Lookup("blue")
if route.Available {
    result := route.Pool.Current().Process("Hello World")
    fmt.Println(result.Text)
}
```

[Lookup](../examples/workers/lookup.go) reads the dispatcher's pool index under a
lock. Callers keep the returned reference and call the pool directly.
Current panics for an empty or unstarted reference.

## Reference operations — exhaustive

| Form | Example |
| --- | --- |
| Start or reserve a reference | `supervise.Start[Worker](group, args)`, `supervise.Reserve[Worker]()` |
| Declare, construct, or allocate an empty reference | `var worker supervise.Ref[Worker]`, `supervise.Ref[Worker]{}`, `new(supervise.Ref[Worker])` |
| Alias the reference type | `type WorkerRef = supervise.Ref[Worker]` |
| Copy, assign, compare, pass, or return concrete references | `next := worker`, `next == worker`, `use(worker)`, `return worker` |
| Store references in variables, fields, arrays, native containers, or frozen collections | `map[string]supervise.Ref[Worker]`, `Route{Worker: worker}` |
| Read, address, dereference, index, slice, or update ordinary reference storage | `p := &worker`, `other := *p`, `workers[name] = worker`, `workers[1:]` |
| Range over native storage or concrete iterators; use concrete switch cases | `for _, worker := range workers { use(worker) }`, `for worker := range iterator { use(worker) }`, `switch worker { case selected: use(worker) }` |
| Use native container operations: make, append, copy, delete, clear, len, cap, close, send, and receive | `make([]supervise.Ref[Worker], 2)`, `append(workers, worker)`, `copy(dst, src)`, `delete(workers, name)`, `clear(workers)`, `len(workers)`, `cap(workers)`, `close(ch)`, `ch <- worker`, `<-ch` |
| Keep an interface with a concrete reference result | `type Producer interface { Worker() supervise.Ref[Worker] }`; call `producer.Worker().Current().Ready()`. |
| Capture references or pass them through concrete callbacks | `func() { worker.Current().Process("Hello") }` |
| Call an exported service method directly; parentheses, go, and defer are allowed | `worker.Current().Process("Hello")`, `go worker.Current().Process("Hello")` |

Type rules:

- `T` must be a struct that embeds `supervise.Service`.
- Fixed im and supervise generic APIs are allowed. User generic type arguments
  cannot contain references, including in fields or callback signatures.
- A generic function can declare a concrete Ref parameter:
  `func use[T any](worker supervise.Ref[Worker], value T)`.
- Ref type conversions and conversion of Ref values to interfaces are rejected.
- A callback with a Ref parameter or result keeps its concrete signature.
  An interface with such a method keeps its exact interface type. Neither can
  be converted to `any` or another interface.
- Marked objects and frozen collections keep all im rules.

Copy a reference from a marked field to a local before capturing it or using it
in go or defer:

```go
pool := route.Pool
go pool.Current().Process("Hello")
```

## Current calls — exhaustive

| Form | Rule |
| --- | --- |
| `ref.Current().Method(args)` | Method is an exported service method, including a promoted method, other than Init, Run, or a method supplied by the runtime. It returns no result or one [service result](service-results.md). |

Current's `*T` result is used only as the receiver of this call. The checker
rejects saving that pointer. Service method bodies use their own receiver and
runtime methods under ordinary Go rules.

## Rejected examples — non-exhaustive

| Rejected | Use instead |
| --- | --- |
| `worker := ref.Current()` | Keep `ref`; call `ref.Current().Ready()` each time. |
| `ref.Current().Count`, `*ref.Current() = Worker{}` | Add an exported service method for the operation. |
| `current := ref.Current`, `method := ref.Current().Process` | Use a closure that captures the reference and calls `ref.Current().Process("Hello")`. |
| `supervise.Ref[Worker].Current(ref)` | `ref.Current().Ready()` |
| `ref.Current().Init()`, `ref.Current().Args()` | Initialize through Start; expose needed data through an exported service method. |
| A function without a body that receives or returns references | Supply a Go body so its reference use can be checked. |
| `var value any = ref`, `identity(ref)` with generic identity | Use a concrete `supervise.Ref[Worker]` parameter. |
| `wrapper.Current()` through an embedded Ref | Select the field: `wrapper.Ref.Current().Ready()`. |

Calls run in the caller's goroutine. Review locks, inputs, and callbacks in the
service implementation. The [reference proof](arch/services.md#service-reference-boundary)
states what the checker proves and which code it trusts.
