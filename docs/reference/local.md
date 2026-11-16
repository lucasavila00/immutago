# Local values and records

Local values are projections, records that embed `im.Local`, and unnamed arrays
of Local values. Local record fields require [read results](types.md). Their
fields are immutable from construction; they need no generated code.

```go
type Prediction struct {
    im.Local
    Name im.StringProjection
    Score float64
}
```

A copy remains Local and can share immutable backing storage. Use `im.Copied`
when every field meets its [value-only field rule](copied.md#fields--exhaustive)
and callers need to store or change independent copies.

## Value forms — exhaustive

These forms apply to Local records, generated projections, and primitive
projections. Combine them with the [common source forms](common-forms.md).
Unlisted operations on protected storage are rejected.

| Allowed form | Example |
| --- | --- |
| Zero declaration or empty literal | `var p Prediction`, `im.StringProjection{}`, `var raw im.Projection[Settings]`, `SettingsProjection{}` |
| Construct a Local record | `p := Prediction{Name: im.ProjectString("first"), Score: 2}` |
| Construct a projection with its fixed API | `p := cell.Load()`, `name := im.ProjectString(text)`; see [generated](projections.md) and [primitive](projection-primitives.md) projection forms. |
| Copy, replace, or discard locally | `other := p`, `p = other`, `_ = p` |
| Read a Local record field | `score := p.Score`, `name := p.Name` |
| Store an UncheckedValue field | `type Reply struct { im.Local; Model im.UncheckedValue[Model] }` |
| Use a field in another Local record | `type Reply struct { im.Local; Item Prediction }` |
| Use a local unnamed array | `values := [2]Prediction{p}` |
| Use an exclusive local native builder | `names := []im.StringProjection{name}`, `names := map[string]im.StringProjection{"first": name}` |
| Pass to a direct function or method with an exact concrete parameter | `read(p)` with `func read(p Prediction)` |
| Return a value through a concrete result | `return p` from `func predict() Prediction` |

Local values cannot enter package variables, Frozen fields, Cells, frozen
entries, or service start arguments. This restriction follows nested fields
and arrays. Interfaces, callbacks, closure captures, user generics, go, and
defer do not transport Local values.

## Pointer forms

Use the complete [common pointer forms](common-forms.md#common-pointer-forms)
for a Local record: for example, `p := &Prediction{}` followed by a direct typed
call. The pointer must stay local. It cannot be stored, returned, or captured.

## Local builders

Local builders use the complete [builder operations](frozen.md#build-before-freezing).
Their keys and elements must be Frozen types or read results; map keys must also
be Go-comparable. A Local value can be a key: `counts := map[im.StringProjection]int{name: 1}`.
The builder handle cannot be copied, passed, returned, captured, or saved.
Frozen collections reject Local keys and elements, so these builders cannot Freeze.

## Rejected examples — non-exhaustive

| Rejected | Allowed replacement |
| --- | --- |
| Save `p` in a package variable | Save `p.Name.Clone()` and `p.Score`. |
| `im.Cell[Prediction]` or a Prediction field in a Frozen record | Store Frozen data and create Local results when reading. |
| `type Reply struct { im.Local; Value Settings }` with Frozen Settings | Use SettingsProjection, or copy its required numeric fields. |
| `callback := func() float64 { return p.Score }` | Copy `score := p.Score` before the closure. |
| `go read(p)` or `defer read(p)` | Call `read(p)` synchronously. |
| `fmt.Sprintf("%s", name)` with StringProjection | Use `name.Unchecked()` for shared string access or `name.Clone()` for independent storage. |
| `return &Prediction{}` | Return a Prediction value. |
| `im.FreezeSlice(names)` for `names []im.StringProjection` | Read the local builder directly; construct independent stored data when needed. |
| `return names` for a native Local builder | Return a Local record with the required projections. |
