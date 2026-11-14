# Shared objects

```go
type SharedPoint struct { im.Shared; X int }
type SharedLink struct { im.Shared; P *SharedPoint }

func makeSharedPoint() *SharedPoint { return &SharedPoint{X: 1} }

func use() {
    link := SharedLink{P: makeSharedPoint()}
    println(link.P.X)
    // link.P.X = 2 // Rejected. Create another SharedPoint instead.
}
```

## Allowed forms

The [common forms](common-forms.md) and the table below give the complete list
of allowed operations. Other protected operations are rejected.

`*SharedPoint` also permits these forms:

| Allowed form | Example |
| --- | --- |
| Store in a marked field | `link := SharedLink{P: p}` |
| Store in a local slice or map before Freeze | `values := []*SharedPoint{p}` |
| Store in a package variable | `saved = p` |
| Read a stored pointer | `p := link.P`, `p := values[i]` |
| Return or receive a pointer from a named function | `return p`, `p := makeSharedPoint()` |

## Rejected examples

These examples are not exhaustive.

| Rejected form | Allowed replacement |
| --- | --- |
| `p.X = 2` | `p = &SharedPoint{X: 2}` |
| `p := &point` | `p := new(point)` |
| `var value any = p` | Keep a concrete `*SharedPoint` variable. |
