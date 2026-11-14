# Shared objects

[Language guide](rules.md) · [Common forms](common-forms.md) · [Scoped](scoped.md)

Use `im.Shared` when you need to save or return a pointer. Set the fields when
you create the object. To change data, replace the pointer with a new object:

```go
type SharedPoint struct { im.Shared; X int }
type SharedLink struct { im.Shared; P *SharedPoint }

func makeSharedPoint() *SharedPoint { return &SharedPoint{X: 1} }

func use() {
    p := makeSharedPoint()
    q := p
    p = &SharedPoint{X: 2} // Allowed. q still points to X == 1.
    _ = q
    // p.X = 3           // Rejected. Fields cannot change.
}
```

## Allowed forms

The [common forms](common-forms.md) and the table below give the complete list
of allowed operations. Other protected operations are rejected.

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
