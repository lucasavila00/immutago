# ByValue objects

`im.ByValue` is an empty struct embedded as a marker for the checker.
Use it when callers need value copies but must not keep pointers to the object:

- Fields are read-only.
- Values can be copied and returned. Values without local views can also be
  stored in marked structs or shared frozen collections.
- Pointers can stay in local variables or pass to direct checked functions and
  [methods](common-forms.md#methods).
- Pointers cannot be stored in fields or containers, saved globally, returned,
  or captured by closures.

The marker does not copy referenced data. A ByValue value can retain storage
through shared frozen values, Shared pointers, service references, or
`im.UncheckedValue` fields.

A ByValue value with a [local frozen view](local-views.md) field has the local
view rules as a whole. This includes nested records and unnamed arrays. Such a
value cannot be saved in a package variable, Shared field, Owner slot, service
start arguments, or shared frozen collection. A local copy or return keeps
this restriction.

## Example: a request limit

Read settings and return the remaining limit:

```go
package main

import "github.com/lucasavila00/immutago/pkg/im"

type Settings struct {
    im.ByValue
    Limit int
}

func remaining(settings *Settings, used int) int {
    return settings.Limit - used
}

func copySettings(settings *Settings) Settings {
    return *settings
}

func main() {
    settings := &Settings{Limit: 100}
    println(remaining(settings, 30)) // 70
    saved := copySettings(settings)
    println(saved.Limit) // 100; the value copy can be kept.
}
```

## Allowed forms — exhaustive

Use the [common forms](common-forms.md).

| Need | Use |
| --- | --- |
| Copy or return the data | `value := *p`, then `return value`. |
| Store data that contains no local views | A marked field: `type Snapshot struct { im.Shared; Settings Settings }`. |
| Return a view in a local record | `type Result struct { im.ByValue; Text im.FrozenByValueString }`. |
| Keep the pointer | A type with [im.Shared](shared.md). |
| Publish a new version for readers | [im.Owner](owner.md). |

Exported [service methods](service-results.md) can return a ByValue struct only
when its fields contain no shared data references. They can return local views,
scalars, service references, and unnamed arrays of allowed results.

## Rejected examples — non-exhaustive

| Rejected form | Allowed replacement |
| --- | --- |
| `return p` with `p *Settings` | `return *p` with a `Settings` result. |
| `saved = p` with `p *Settings` | Store the value copy `*p`. |
| `callback := func() int { return p.Limit }` | Read `x := p.Limit` before creating the callback. |
| `slot.Store(p)` | `slot.Store(&Settings{Limit: p.Limit})` |
| Save a Result containing a local view in a package variable | Clone its view fields and save independent data. |
