# Scoped objects

[Language guide](rules.md) · [Common forms](common-forms.md) · [Owner](owner.md)

Use `im.Scoped` when a pointer must stay local. You can pass it to a named
function with the exact pointer parameter type. That function must not save
or return the pointer:

```go
type ScopedPoint struct { im.Scoped; X int }

func read(p *ScopedPoint) int { return p.X }

func use() int {
    p := &ScopedPoint{X: 1}
    return read(p)
}
```

## Allowed forms

The [common forms](common-forms.md) give the complete list of allowed operations.
Scoped does not add pointer storage or return forms. Other protected operations
are rejected.

A value copy can be stored or returned: `value := *p`, then `return value`.
To publish a fresh Scoped object, use the explicit [Owner forms](owner.md).
An existing pointer cannot be passed to `Owner.Store`.

## Rejected examples

These examples are not exhaustive.

| Rejected form | Allowed replacement |
| --- | --- |
| `return p` with `p *ScopedPoint` | `return *p` with a `ScopedPoint` result. |
| `saved = p` with `p *ScopedPoint` | Store the value copy `*p`. |
| `callback := func() int { return p.X }` | Read `x := p.X` before creating the callback. |
| `slot.Store(p)` | `slot.Store(&ScopedPoint{X: p.X})` |

Use [Shared](shared.md) when pointers must be stored or returned.
