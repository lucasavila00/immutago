# Scoped objects

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
A Scoped pointer cannot be saved in a field or package variable, or returned
from a function. Operations not listed in the common forms are rejected.

A value copy can be stored or returned: `value := *p`, then `return value`.
To let other code read the current Scoped object, use [Owner](owner.md).
An existing pointer cannot be passed to `Owner.Store`.

## Rejected examples

These examples are not exhaustive.

| Rejected form | Allowed replacement |
| --- | --- |
| `return p` with `p *ScopedPoint` | `return *p` with a `ScopedPoint` result. |
| `saved = p` with `p *ScopedPoint` | Store the value copy `*p`. |
| `callback := func() int { return p.X }` | Read `x := p.X` before creating the callback. |
| `slot.Store(p)` | `slot.Store(&ScopedPoint{X: p.X})` |
