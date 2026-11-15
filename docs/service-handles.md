# Service handles

`supervise.Ref[T]` identifies a service. Copy, store, or return the handle; call
its exported message methods with `ref.Current().Method(args)`.
A restart changes the service instance, not the handle's identity.

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

[Lookup](../examples/workers/lookup.go) reads the dispatcher's pool index under a
lock. Callers keep the returned handle and call the pool directly.
Use a started handle: Current panics for an empty or unstarted handle.

## Handle operations — exhaustive

| Form | Example |
| --- | --- |
| Start, reserve, or declare an empty handle | `supervise.Start[Worker](group, args)`, `supervise.Reserve[Worker]()`, `var worker supervise.Ref[Worker]` |
| Alias the handle type | `type WorkerRef = supervise.Ref[Worker]` |
| Copy, assign, compare, pass, or return concrete handles | `next := worker`, `next == worker`, `use(worker)`, `return worker` |
| Store handles in variables, fields, arrays, native containers, or frozen collections | `map[string]supervise.Ref[Worker]`, `Route{Worker: worker}` |
| Read, address, index, slice, range over, or update ordinary handle storage | `workers[name] = worker`, `for _, worker := range workers { use(worker) }` |
| Use native container operations | `append(workers, worker)`, `delete(workers, name)`, `ch <- worker`, `<-ch` |
| Capture handles or pass them through concrete callbacks | `func() { worker.Current().Resize(image) }` |
| Call a message directly; parentheses, go, and defer are allowed | `worker.Current().Resize(image)` |

`T` must embed `supervise.Service`. Fixed `im` and `supervise` generic APIs are
allowed. User generics, type conversions, and interfaces cannot transport
handles, including inside other types. Marked fields and frozen collection
storage retain their usual write restrictions.

## Current view — exhaustive

| Form | Rule |
| --- | --- |
| `ref.Current().Message(args)` | Message is an exported user method, including a promoted method, other than Init or Run. It returns nothing or one [Scoped result](scoped-results.md). |

This is a method-only view. The Go `*T` return type does not permit other uses in
checked code. Service methods use their own receiver and runtime methods under
ordinary Go rules.

## Rejected examples — non-exhaustive

| Rejected | Use instead |
| --- | --- |
| `worker := ref.Current()` | Keep `ref`; call `ref.Current().Message()` each time. |
| `ref.Current().Count`, `*ref.Current() = Worker{}` | Add a message method for the operation. |
| `current := ref.Current`, `method := ref.Current().Message` | Use a closure that captures the handle and makes the direct message call. |
| `supervise.Ref[Worker].Current(ref)` | `ref.Current().Message()` |
| `ref.Current().Init()`, `ref.Current().Args()` | Initialize through Start; expose needed data through a message. |
| `var value any = ref`, `identity(ref)` with generic identity | Use a concrete `supervise.Ref[Worker]` parameter. |
| `wrapper.Current()` through an embedded Ref | Select the handle field: `wrapper.Ref.Current().Message()`. |

The handle does not make service state immutable or serialize calls. Review
locks, inputs, and callbacks in the service implementation. The
[soundness argument](arch/ownership.md#service-handle-boundary) states what the
checker proves and which code it trusts.
