# Local records

Embed `im.Local` to combine local read projections with copied values.
Local records need no generation:

```go
type Prediction struct {
    im.Local
    Name im.StringProjection
    Score float64
}

func prediction(name string, score float64) Prediction {
    return Prediction{Name: im.Text(im.ProjectionOf(name)), Score: score}
}
```

Its fields must be [read results](types.md). Copying the record keeps the local
restriction. It does not copy the name bytes.

| Allowed forms — exhaustive | Example |
| --- | --- |
| Construct, copy, replace, or discard a local value | `p := prediction("first", 2)`, `other := p`, `_ = other` |
| Read a field | `score := p.Score`, `name := p.Name` |
| Put in another Local record or local unnamed array | `[2]Prediction{p}` |
| Put in an exclusive local native builder | `values := []im.StringProjection{name}`, `names := map[string]im.StringProjection{"first": name}` |
| Pass to a direct concrete function or method | `read(p)` with `func read(p Prediction)` |
| Return through a concrete result | `return p` |
| Create a local pointer for a direct typed call | `p := &Prediction{}`; use the [pointer forms](common-forms.md#common-pointer-forms). |

A Local pointer cannot be stored, returned, or captured. Local values also
cannot enter globals, Frozen fields, Cells, shared frozen entries, or service
start arguments. Interfaces, callbacks, closure captures, user generics, go,
and defer do not transport them.

Local builders use the [fixed builder operations](frozen.md#build-before-freezing).
Their keys and elements must be Frozen types or read results; map keys must also
be Go-comparable. A Local value can be a key: `counts := map[im.StringProjection]int{name: 1}`.
The builder handle cannot be copied, passed, returned, captured, or saved. It
cannot be frozen because Frozen collections reject Local keys and elements.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| Save `p` in a package variable | Save `p.Name.Clone()` and `p.Score`. |
| `im.Cell[Prediction]` | Store Frozen data and return a Local result when reading it. |
| `callback := func() float64 { return p.Score }` | Copy `score := p.Score` before the closure. |
| `return &Prediction{}` | Return a Prediction value. |
| `im.FreezeSlice(values)` for `values []im.StringProjection` | Read the local builder directly; make independent stored data when needed. |
| `return values` for a native Local builder | Return a Local record with the required projections. |
