# Type rules

These categories define fields, Cells, projections, and service results.

| Category — exhaustive | Types |
| --- | --- |
| Frozen | Booleans, numbers, native strings, FrozenString, unnamed arrays of Frozen values, Frozen records, pointers to Frozen records, FrozenSlice/FrozenMap with Frozen arguments, UncheckedValue handles, and service references. |
| Copied | Booleans, numbers, service references, and unnamed arrays of Copied values. |
| Local | Projection, StringProjection, SliceProjection, MapProjection, Local records, and unnamed arrays whose elements are Local. |
| Read result | Copied or Local. |

Defined scalar types and aliases keep their category. Named arrays need a
marked struct wrapper. Copied excludes native strings, pointers, and all structs
except opaque service references. Frozen records are not Copied.
Copying a service reference does not copy its service.
Embed a marker to declare a record category. The markers themselves are not
Frozen, Copied, or read-result payload types.

## Markers — exhaustive

| Marker | Required fields | Record use |
| --- | --- | --- |
| `im.Frozen` | Frozen fields | Can be copied, stored, and returned from ordinary functions. |
| `im.Local` | Read-result fields | Can be copied locally, passed directly, and returned; cannot enter persistent storage. |

- Embed exactly one marker by value; aliases work.
- Declare Frozen records as named package-scope types in non-test Go files.
  Local records need no generation.
- Set fields at construction. Later field, element, or dereference writes fail.
- Each nested record needs its own marker. Unmarked structs cannot own
  protected data.
- Native containers, interfaces, functions, and Local values are not Frozen fields.
- A Cell is mutable storage, not a Frozen value or read result.

## API requirements — exhaustive

| API | Type requirement |
| --- | --- |
| `Cell[T]`, `Projection[T]`, `ProjectionOf(value)` | T is a Frozen type. |
| `FrozenSlice[E]`, `FrozenMap[K,V]` | Each argument is a Frozen type; keys are also Go-comparable. |
| `ProjectString(value)` | A native string type. |
| `frozen.Projection()` | FrozenString, FrozenSlice[E], or FrozenMap[K,V]. |
| [Generic projection adapters](projection-primitives.md#projection-operations--exhaustive) | A Projection of the matching string, FrozenString, FrozenSlice, or FrozenMap type. |
| `IsNil(Projection[*T])` | T is a Frozen record. |
| `Copy(Projection[T])` | T is Copied. |
| `CopyRecord(Projection[T])` | T is a concrete Frozen record; returns a projection of a new copied root. |
| `Cell[T].LoadPointer()`, `SliceProjection[T].AtPointer(i)` | T is a concrete Frozen record; the result is Projection[*T]. |
| `SliceProjection[E].Clone()` | E is Copied. |
| `MapProjection[K,V].Clone()` | K and V are Copied. |
| Exported service method result | No result or one read result. |

The service rule does not change a struct's field rules. A Frozen record can be
stored in a Cell or read through Projection. A service returns its generated
projection or an im.Local result, including when all fields are numbers.

## Rejected examples — non-exhaustive

| Rejected | Use instead |
| --- | --- |
| `struct { Mark im.Frozen }` or `struct { *im.Frozen }` | Embed `im.Frozen` by value. |
| `im.Projection[im.Frozen]` | Embed the marker in a record and use that record. |
| `struct { im.Frozen; Text im.StringProjection }` | Use `im.Local` for a local result. |
| `struct { im.Local; Text string }` | Use `im.StringProjection`. |
| `struct { im.Local; Text im.FrozenString }` | Use `im.StringProjection`. |
| `struct { im.Local; Count FrozenCount }`, where FrozenCount embeds im.Frozen | Use `FrozenCountProjection`, or read its numeric fields into a Local record. |
| `type Counts [2]int` as protected storage | `type Counts struct { im.Frozen; Values [2]int }`. |
| `im.Copy(textProjection)` with a string or FrozenString payload | Use the matching [projection adapter](projection-primitives.md#projection-operations--exhaustive), then Unsafe for shared access or Clone for independent storage. |
| `im.Copy(countProjection)` with `countProjection im.Projection[FrozenCount]` | Use a generated projection, or copy individual numeric fields. |
| Save a Local record in a package variable | Copy the required data into independent storage. |
