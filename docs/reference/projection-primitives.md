# Projection primitives

These fixed APIs build the [generated typed projections](projections.md). They use the same
[type rules](types.md) and source checks as other application code.

## Projection operations — exhaustive

| Operation | Requirement and result |
| --- | --- |
| `im.ProjectString(value)` | Native string input; returns StringProjection. |
| `frozen.Projection()` | FrozenString, FrozenSlice[E], or FrozenMap[K,V] receiver; returns StringProjection, SliceProjection[E], or MapProjection[K,V]. |
| `im.ProjectionOf(value)` | Frozen input; returns Projection[T]. |
| `projection.Unchecked()` | Requires Projection[T] with Frozen T; returns T by value with its shared storage. A pointer result is the same pointer. |
| `im.CopyRecord(value)` | Requires Projection[T] for a concrete Frozen record T; copies it into a new root and returns Projection[*T]. |
| `im.Value(projection)` | T must be Frozen and a [read result](types.md#read-results--exhaustive); returns T. |
| `im.StringOf(projection)` | Requires Projection[S] with underlying string S; returns StringProjection. |
| `im.StringOfFrozen(projection)` | Requires Projection[FrozenString]; returns StringProjection. |
| `im.SliceOf(projection)` | Requires Projection[FrozenSlice[E]]; returns SliceProjection[E]. |
| `im.MapOf(projection)` | Requires Projection[FrozenMap[K,V]]; returns MapProjection[K,V]. |
| `im.IsNil(projection)` | Requires Projection[*T] for a Frozen record T; reports nil without exposing the pointer. |

Application code uses the listed Value, Unchecked, and adapter forms. Unchecked permits
retention; it does not change the payload's type rules or permit target writes.
Raw slot reads use the [Cell methods](cell.md#raw-cell-methods--exhaustive).

## Collection projections — exhaustive

```go
scores := im.FreezeMap(map[string]int{"alice": 10})
projection := scores.Projection()
value, found := projection.Get("alice")
if found { println(im.Value(value)) }
for key, value := range projection.All() {
    println(im.StringOf(key).Unchecked(), im.Value(value))
}
```

| Operation | StringProjection | SliceProjection[E] | MapProjection[K,V] |
| --- | --- | --- | --- |
| Len | Bytes | Elements | Entries |
| At | Byte | Projection[E], by value | — |
| AtPointer | — | Projection[*E]; E must be a concrete Frozen record | — |
| Equal | Compare with StringProjection | — | — |
| Is | Compare with native string | — | — |
| Unchecked | Shared native string | — | — |
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

## Storage and calls

Use the complete [Local value forms](local.md#value-forms--exhaustive) for
construction and transport, and the [common forms](common-forms.md) for statement
contexts. Unlisted operations on protected storage are rejected.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `im.Value(im.ProjectionOf("text"))` | Use `im.ProjectString("text")`; use Unchecked for a shared native string or Clone for independent storage. |
| `im.Value(record)` with `record im.Projection[Settings]`, including a record with only numeric fields | Use a generated projection for local field reads, or `record.Unchecked()` for a shared Settings value. |
| Clone a projection of a string-key map | Iterate; Clone each key and copy each value into a new map. |
| Clone a slice or map projection with Frozen record elements | Read record projections and copy the required fields into new data. |

The caller controls retention of an Unchecked result and must not give a
returned pointer to an API that changes its target. UncheckedValue keeps its
[package access rules](unchecked-values.md). See [operation costs](../../pkg/im/costs.md)
for allocation and copy costs.
