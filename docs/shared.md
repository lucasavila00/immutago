# Shared objects

[Language guide](rules.md) · [Common forms](common-forms.md) · [Scoped](scoped.md)

Use `im.Shared` when pointers to immutable objects must be stored or returned.
The object is immutable from construction; a variable holding its pointer can
be reassigned.

```go
type SharedPoint struct { im.Shared; X int }
type SharedLink struct { im.Shared; P *SharedPoint }

func makeSharedPoint() *SharedPoint { return &SharedPoint{X: 1} }
```

## Allowed forms

Use the common lists for [stored types](common-forms.md#markers-and-stored-types),
[construction and reads](common-forms.md#construction-and-scalar-reads),
[pointers](common-forms.md#common-pointer-forms), and
[statements and calls](common-forms.md#statements-and-calls).
Those lists and the table below are complete. Other protected operations are rejected.

`*SharedPoint` also permits these forms:

| Allowed form | Example |
| --- | --- |
| Store in a marked field | `link := SharedLink{P: p}` |
| Store in an exclusive native builder | `values := []*SharedPoint{p}` |
| Store in a package variable | `saved = p` |
| Read a stored pointer | `p := link.P`, `p := values[i]` |
| Return or receive a pointer from a named function | `return p`, `p := makeSharedPoint()` |

## Rejected examples

These examples are not exhaustive. The [common forms](common-forms.md) contain
more rejections and replacements.

| Rejected form | Allowed replacement |
| --- | --- |
| `p.X = 2` | `p = &SharedPoint{X: 2}` |
| `p := &point` | `p := new(point)` |
| `var value any = p` | Keep a concrete `*SharedPoint` variable. |

Use [Scoped](scoped.md) when pointer retention must be restricted.
