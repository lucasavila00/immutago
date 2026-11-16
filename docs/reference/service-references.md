# Service references

`supervise.Ref[T]` identifies one service. Copy, store, or return the reference.
Call an exported service method with `ref.Borrow().Method(args)`.
Copies refer to the same service.

## Optional references

The zero `Ref` is the unset value. Use `IsZero` to test an optional reference:

```go
type optionalRoute struct{ cache supervise.Ref[Counter] }

func optionalCount(r optionalRoute) int {
    if !r.cache.IsZero() {
        return r.cache.Borrow().Count()
    }
    return 0
}
```

`IsZero` tests whether Start supplied a reference. A successful Start returns
a nonzero reference, which stays nonzero after stop. Copies give the same result.

`IsZero` uses ordinary reference call forms. Direct calls also work on a Ref
field or array element in a Frozen or Local record:

```go
if !route.Worker.IsZero() {
    route.Worker.Borrow().Count()
}
```

Select an embedded Ref field explicitly. The [common forms](common-forms.md)
apply to the record that holds it:

| Use a Ref field | Frozen record | Local record |
| --- | --- | --- |
| `check := route.Worker.IsZero` | Allowed; copies the Ref. | Allowed; copies the Ref. |
| `go route.Worker.IsZero()` or `defer route.Worker.IsZero()` | Allowed. | Copy the Ref first. |
| `func() bool { return route.Worker.IsZero() }` | Allowed. | Copy the Ref first; do not capture route. |

## Checked operations — exhaustive

| Operation | Rule |
| --- | --- |
| Declare or use `supervise.Ref[T]` | T is a concrete struct with a [value embedding path](services.md#service-embedding--exhaustive) to `supervise.Service[A]`. |
| Call the concrete Ref's Borrow method | Use `ref.Borrow().Method(args)`; see the call forms below. |
| Return data from an exported service method | Use the [service result signatures](service-results.md). |
| Use a reference inside protected im storage | Apply the [im rules](common-forms.md) to that storage. |

Reference storage and transport use ordinary Go rules. This includes copies,
containers, pointers, conversions, interfaces, generic helpers, callbacks,
`go`, `defer`, and functions implemented outside Go. A field containing a Ref
does not make its holder protected storage.

| Ordinary Go examples | Form |
| --- | --- |
| Store or copy a reference | `type Server struct { Worker supervise.Ref[Worker] }`, `other := ref` |
| Use a pointer, generic helper, or interface | `p := &ref`, `other := identity(ref)`, `var value any = ref` |
| Use atomic storage | `var selected atomic.Pointer[supervise.Ref[Worker]]`; `selected.Store(&ref)` |

Protected storage still has its own rules. For example, a Ref field in a Frozen
record cannot be replaced. Copy the field into an ordinary variable to store or
pass the reference independently.

## Borrow calls — exhaustive

| Form | Rule |
| --- | --- |
| `ref.Borrow().Method(args)` with a concrete Ref receiver | Method is an exported service method, including a promoted method, other than Init, Run, or a method supplied by the runtime. Its results follow the [service result signatures](service-results.md). |
| `selected.Borrow()` through an interface or type parameter, including method values and expressions | Ordinary Go rules apply: `pointer := selected.Borrow()`, `accessor := selected.Borrow`, `Borrower.Borrow(selected)` where Borrower is the interface type. The result can be stored and used outside the checked call form. |

For direct Ref calls, the checker rejects every other use of `Borrow`, despite
its Go return type of `*T`. Borrow panics with `supervise: zero Ref` for a zero
reference.
Each call returns the service created by Start. Stop does not invalidate the
reference or wait for these method calls.
Borrow is shared and is not exclusive. No release call is needed.

Interfaces and generic methods can expose the service pointer outside this
checked call form. This is accepted, including when an embedded Ref promotes Borrow:

```go
var selected interface{ Borrow() *Worker } = ref
pointer := selected.Borrow()
```

The direct Ref call rule does not apply to interface or type-parameter method
calls. The pointer still refers to that service after Stop.

## Rejected examples — non-exhaustive

| Rejected | Use instead |
| --- | --- |
| `worker := ref.Borrow()` | Keep `ref`; call `ref.Borrow().Method()` each time. |
| `ref.Borrow().Count`, `*ref.Borrow() = Worker{}` | Add an exported service method for the operation. |
| `current := ref.Borrow`, `method := ref.Borrow().Method` | Use a closure that captures the reference and makes the direct method call. |
| `supervise.Ref[Worker].Borrow(ref)` | `ref.Borrow().Method()` |
| `ref.Borrow().Init()`, `ref.Borrow().Run(ctx)`, `ref.Borrow().Args()` | Initialize through Start; expose needed data through an exported service method. |
| `go route.Worker.IsZero()` or `defer route.Worker.IsZero()` where route is Local | Copy the field first: `ref := route.Worker`, then `go ref.IsZero()` or `defer ref.IsZero()`. |
| `route.IsZero()` through an embedded Ref in a marked object | Select the field: `route.Ref.IsZero()`. |
| `wrapper.Borrow()` through an embedded Ref | Select the reference field: `wrapper.Ref.Borrow().Method()`. |

## What the reference rule does not check

- Service fields can change. A Ref field in an immutable object cannot change,
  but the selected service can.
- Calls run in the caller's goroutine. They can run at the same time as Run and
  other calls. The service must protect shared mutable fields.
- A service method can give a callback or global access to its mutable state.
  The reference rule does not prevent this.
