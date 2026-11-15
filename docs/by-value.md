# ByValue objects

`im.ByValue` is an empty struct embedded as a marker for the checker.
Use it when callers can keep value copies but must not keep pointers to the object:

- Fields are read-only.
- Values can be copied, stored in marked structs or frozen collections, and returned.
- Pointers can stay in local variables or pass to direct checked functions and
  [methods](common-forms.md#methods).
- Pointers cannot be stored in fields or containers, saved globally, returned,
  or captured by closures.

ByValue is a rule about pointer use. It does not tie an object to a Go block,
make a value expire when a function returns, or set a garbage-collection deadline.
Value copies can retain memory through strings, frozen collections, Shared
pointers, or `im.UncheckedValue` fields.

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
| Store the data in another object | A marked field: `type Snapshot struct { im.Shared; Settings Settings }`. |
| Keep the pointer | A type with [im.Shared](shared.md). |
| Publish a new version for readers | [im.Owner](owner.md). |

Exported [service methods](service-results.md) can return a ByValue value. Their
result rule also permits scalars and unnamed arrays; it does not change this
marker's pointer rules.

## Rejected examples — non-exhaustive

| Rejected form | Allowed replacement |
| --- | --- |
| `return p` with `p *Settings` | `return *p` with a `Settings` result. |
| `saved = p` with `p *Settings` | Store the value copy `*p`. |
| `callback := func() int { return p.Limit }` | Read `x := p.Limit` before creating the callback. |
| `slot.Store(p)` | `slot.Store(&Settings{Limit: p.Limit})` |
