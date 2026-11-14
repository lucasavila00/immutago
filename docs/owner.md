# Owner publication slots

[Language guide](rules.md) · [Scoped](scoped.md) · [Common forms](common-forms.md)

Use `im.Owner[T]` to publish a new Scoped object and read the current one.
`Store` replaces the current pointer. It does not change an object that a reader
already loaded.

```go
type ScopedPoint struct { im.Scoped; X int }
type Loader struct { Current im.Owner[ScopedPoint] }

func use() int {
    loader := &Loader{}
    loader.Current.Store(&ScopedPoint{X: 1})
    point := loader.Current.Load()
    return point.X
}
```

`Load` returns nil before the first `Store`. Both calls use atomic pointer
operations. An earlier loaded pointer remains valid after a later `Store`.

## Allowed forms

An Owner field or variable is a **slot**. A struct with a slot field, such as
`Loader`, is a **holder**. These allowed forms are complete; other slot and
holder operations are rejected.
Parentheses preserve these forms. Exported package slots can also use a
package qualifier.

| Allowed form | Example |
| --- | --- |
| Named Owner field in a struct without an im marker | `type Loader struct { Current im.Owner[ScopedPoint] }` |
| Declare a package Owner variable without an initializer | `var current im.Owner[ScopedPoint]` |
| Declare a local or package holder variable without an initializer | `var loader Loader` |
| Create a holder and keep its pointer in a local variable | `loader := &Loader{}` or `loader := new(Loader)` |
| Copy a holder pointer to one local variable | `other := loader` |
| Pass a holder pointer to a direct, concrete function | `publish(loader)` where `func publish(l *Loader)` |
| Store a fresh object in a direct slot field or package slot | `loader.Current.Store(&ScopedPoint{X: 1})`, `current.Store(new(ScopedPoint))` |
| Load into one local pointer | `point := loader.Current.Load()`, `var point = current.Load()`, `point = current.Load()` |

A named function can publish in a background goroutine:

```go
func publish(l *Loader) { l.Current.Store(&ScopedPoint{X: 1}) }

func start() {
    loader := &Loader{}
    go publish(loader)
}
```

A holder is an unmarked struct with a direct named slot field. The slot fields
and zero holder variables in the table are allowed. Slot and holder values
cannot be copied.

Holder pointers such as `*Loader` cannot be stored in fields, containers, or
package variables, returned, or captured. Calls with them require direct named
non-generic functions and fixed, concrete parameters. Explicit `*im.Owner[T]`
values and direct `go` or `defer` slot calls are unsupported.

A loaded pointer follows all [Scoped pointer rules](scoped.md). Load it into a
local before use. A Store argument must be the direct `&T{...}` or `new(T)`
construction for the slot's exact `T`.

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
