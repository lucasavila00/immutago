# Scoped objects

Use `im.Scoped` when a function needs to read an object but must not save its
pointer for later. This is useful for a configuration version loaded from an
[Owner](owner.md): each reader uses it for one call.

The function below can read the settings and return a number. It cannot save
`settings` in a field, package variable, or closure, or return that pointer.
The checker enforces these restrictions in the called function too.

```go
package main

import "github.com/lucasavila00/immutago/pkg/im"

type Settings struct {
    im.Scoped
    Limit int
}

func remaining(settings *Settings, used int) int {
    return settings.Limit - used
}

func main() {
    settings := &Settings{Limit: 100}
    println(remaining(settings, 30)) // 70
}
```

A copy of `Settings` can be stored or returned; its pointer cannot. Choose
[Shared](shared.md) when callers need to keep the pointer. These rules limit
where pointers can be saved; they do not set a deadline for garbage collection.
The reference below uses `ScopedPoint` for a struct with `im.Scoped` and `X int`.

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
