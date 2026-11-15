# Projection primitives

These fixed APIs build the [generated typed projections](projections.md). They use the same
[type rules](types.md) and source checks as other application code.

## Projection operations — exhaustive

| Operation | Requirement and result |
| --- | --- |
| `im.ProjectionOf(value)` | Frozen input; returns Projection[T]. |
| `rawCell.Load()` | Copies T into Projection[T]; reads zero T if the raw slot is empty. |
| `rawCell.LoadPointer()` | Requires a concrete Frozen record T; returns Projection[*T] without copying T. |
| `im.CopyProjection(value)` | Requires Projection[T] for a concrete Frozen record T; copies it into a new root and returns Projection[*T]. |
| `im.Copy(projection)` | Requires Copied T; returns T. |
| `im.String(projection)` | Requires FrozenString; returns StringProjection. |
| `im.Text(projection)` | Requires a string type; returns StringProjection. |
| `im.Slice(projection)` | Requires FrozenSlice[E]; returns SliceProjection[E]. |
| `im.Map(projection)` | Requires FrozenMap[K,V]; returns MapProjection[K,V]. |
| `im.IsNil(projection)` | Requires a pointer payload; reports nil without exposing the pointer. |

Application code uses the listed Copy and adapter forms. The internal Read
operation is restricted to verified generated code. UncheckedValue remains
opaque through a Projection.

## Collection projections — exhaustive

```go
scores := im.FreezeMap(map[string]int{"alice": 10})
projection := im.Map(im.ProjectionOf(scores))
value, found := projection.Get("alice")
if found { println(im.Copy(value)) }
for key, value := range projection.All() {
    println(im.Text(key).Unsafe(), im.Copy(value))
}
```

| Operation | StringProjection | SliceProjection[E] | MapProjection[K,V] |
| --- | --- | --- | --- |
| Len | Bytes | Elements | Entries |
| At | Byte | Projection[E], by value | — |
| AtPointer | — | Projection[*E]; E must be a concrete Frozen record | — |
| Equal | Compare with StringProjection | — | — |
| EqualString | Compare with native string | — | — |
| Unsafe | Shared native string | — | — |
| Get | — | — | Projection[V], bool |
| All | — | Index and Projection[E] | Projection[K] and Projection[V] |
| Clone | Independent native string | Native slice; E must be Copied | Native map; K and V must be Copied |

- Consume All with direct range. Do not save the iterator.
- String equality uses native comparison without allocation; Clone is not needed.
- Get accepts the map's key type. Returned keys always remain projections.
- Clone does not recursively copy arbitrary records. For text keys or nested
  collections, read each value and explicitly build independent data.
- There are no substring projections or writable references. Slice and map projections have
  no native storage getters.
- Methods use exact library receiver types, not promoted methods or method values.

## Local transport — exhaustive

| Form | Example |
| --- | --- |
| Zero declaration or empty literal | `var v im.Projection[Settings]`, `im.StringProjection{}` |
| Local declaration, copy, replacement, or discard | `v := cell.Load()`, `other := v`, `_ = other` |
| Local record field or unnamed array | `type Reply struct { im.Local; Title im.StringProjection }` |
| Exclusive local native builder | `values := []im.StringProjection{text}`; use the [Local builder forms](local.md). |
| Direct concrete argument or result | `func read(v im.StringProjection) int`, `return v` |

These rules follow nested fields and arrays. Projections cannot enter package
variables, Frozen fields, Cells, shared frozen entries, service start arguments,
interfaces, callbacks, closures, user generics, go, or defer.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `raw.Read().Limit` with `raw im.Projection[*Settings]` | `NewSettingsProjection(raw).Limit()` |
| `im.ProjectionOf(1).Read()` | `im.Copy(im.ProjectionOf(1))` |
| `im.Copy(im.ProjectionOf("text"))` | Use StringProjection; use Unsafe for a shared native string or Clone for independent storage. |
| Save a projection in a global | Clone or Copy the data to keep. |
| Clone a projection of a string-key map | Iterate; Clone each key and copy each value into a new map. |
| Capture a projection in a closure | Extract Copied fields or independent copies first. |

Projection creation copies T, not its backing storage. Large records therefore have
ordinary Go value-copy costs. Native strings and FrozenString can share source
bytes. StringProjection.Unsafe returns those bytes as a native string; Clone copies them.
