# Owner publication slots

[Language guide](rules.md) · [Scoped](scoped.md) · [Common forms](common-forms.md)

Use `im.Owner[T]` to publish one current object. It holds a pointer to a concrete,
valid `im.Scoped` struct. The slot is mutable; the object is immutable. `Load` returns nil
before the first `Store`. `Store` and `Load` use atomic pointer operations.
An earlier loaded pointer remains valid after a later `Store`. Go does not
specify when an unused object is collected.

## Allowed forms

These allowed forms are complete. Other slot and holder operations are rejected.
Parentheses preserve these forms. Exported package slots can also use a
package qualifier.

| Allowed form | Example |
| --- | --- |
| Named slot field in an unmarked struct | `type Loader struct { Current im.Owner[ScopedPoint] }` |
| Zero package slot | `var current im.Owner[ScopedPoint]` |
| Zero holder local or package variable | `var loader Loader` |
| Fresh holder pointer in a local | `loader := &Loader{}` or `loader := new(Loader)` |
| Copy a holder pointer to one local | `other := loader` |
| Pass a holder pointer to a direct, concrete function | `publish(loader)` where `func publish(l *Loader)` |
| Store a fresh object in a direct slot field or package slot | `loader.Current.Store(&ScopedPoint{X: 1})`, `current.Store(new(ScopedPoint))` |
| Load into one local pointer | `point := loader.Current.Load()`, `var point = current.Load()`, `point = current.Load()` |

A named function can publish in a background goroutine:

```go
func publish(l *Loader) { l.Current.Store(&ScopedPoint{X: 1}) }
go publish(loader)
```

A holder is an unmarked struct with a direct named slot field. The slot fields
and zero holder variables in the table are allowed. Slot and holder values
cannot be copied.

Holder pointers such as `*Loader` cannot be stored in fields, containers, or
package variables, returned, or captured. Calls with them require direct named
non-generic functions and fixed, concrete parameters. Explicit `*im.Owner[T]`
values and direct `go` or `defer` slot calls are unsupported.

A loaded pointer follows all Scoped pointer rules. Load it into a local before
use. A Store argument must be the direct
`&T{...}` or `new(T)` construction for the slot's exact `T`.

## Rejected examples

These examples are not exhaustive.

| Rejected form | Allowed replacement |
| --- | --- |
| `im.Owner[SharedPoint]` | Use `im.Owner[ScopedPoint]`. |
| `copy := loader.Current` or `copy := *loader` | Keep and pass `loader` as a pointer. |
| `loader.Current.Store(point)` | `loader.Current.Store(&ScopedPoint{X: point.X})` |
| `return loader.Current.Load()` | `point := loader.Current.Load(); return *point` with a value result. |
| `retain(loader)` where `retain` takes `...*Loader` | Use a direct function with a fixed `*Loader` parameter. |
| `var saved *Loader` at package scope | Keep the pointer in a local variable. |
| `read := loader.Current.Load` | `point := loader.Current.Load()` |
| `loader.Current.Load().X = 2` | `loader.Current.Store(&ScopedPoint{X: 2})` |
| `type Outer struct { Value Loader }` | Keep the slot field in the unmarked holder. |
