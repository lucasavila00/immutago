# Scoped objects

`im.Scoped` is an empty struct embedded as a marker for the checker.
Use it for objects that functions can read without keeping their pointers:

- Fields are read-only.
- Functions and [methods](common-forms.md#methods) can read fields and return values.
- They cannot store the pointer in a field or global, capture it, or return it.

For example, read settings and return the remaining limit:

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

## Allowed forms — exhaustive

Use the [common forms](common-forms.md).

| Need | Use |
| --- | --- |
| Store or return the data | A value copy: `value := *p`, then `return value`. |
| Keep the pointer | A type with [im.Shared](shared.md). |
| Publish a new version for readers | [im.Owner](owner.md). |

Scoped limits where pointers can be stored. It does not set a garbage-collection deadline.

## Rejected examples — non-exhaustive

| Rejected form | Allowed replacement |
| --- | --- |
| `return p` with `p *Settings` | `return *p` with a `Settings` result. |
| `saved = p` with `p *Settings` | Store the value copy `*p`. |
| `callback := func() int { return p.Limit }` | Read `x := p.Limit` before creating the callback. |
| `slot.Store(p)` | `slot.Store(&Settings{Limit: p.Limit})` |
