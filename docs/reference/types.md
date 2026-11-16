# Type rules

These categories define fields, Cells, projections, and service data results.
The [service result signatures](service-results.md) also permit an ordinary error;
that error does not belong to these categories.

| Category — exhaustive | Types |
| --- | --- |
| Frozen | Booleans, numbers, native strings, FrozenString, unnamed arrays of Frozen values, Frozen and Copied records, pointers to Frozen records, [Frozen interfaces](frozen-interfaces.md), FrozenSlice/FrozenMap with Frozen arguments, UncheckedValue handles, and service references. |
| Copied | Booleans, numbers, Copied records, service references, and unnamed arrays of Copied values. |
| Local | Projection, StringProjection, SliceProjection, MapProjection, Local records, and unnamed arrays whose elements are Local. |
| Read result | Copied, Local, UncheckedValue handles, and unnamed arrays of read results. |

Defined scalar types and aliases keep their category. Named arrays need a
marked struct wrapper in these categories. Copied excludes native strings and
pointers. Its structs require `im.Copied`, except opaque service references.
Frozen records are not Copied.
Copying a service reference does not copy its service.
Embed a marker to declare a record category. The markers themselves are not
Frozen, Copied, or read-result payload types.

## Markers — exhaustive

| Marker | Required fields | Record use |
| --- | --- | --- |
| `im.Frozen` | Frozen fields | Can be copied, stored, and returned from ordinary functions. |
| `im.Local` | Read-result fields | Can be copied locally, passed directly, and returned; cannot enter persistent storage. |
| `im.Copied` | Boolean/numeric fields, unnamed arrays, and other Copied records; no strings or reference handles | Can be stored by value in ordinary holders and returned by services; ordinary copies can change independently. |

- Embed exactly one marker by value; aliases work.
- Frozen records embed only their marker. Give every data field a name:
  `Count Count`, not an embedded `Count`. Local and Copied field rules do not change.
- Declare Frozen records as named package-scope types in non-test Go files.
  Local and Copied records need no generation.
- Set Frozen and Local fields at construction. Later field, element, or
  dereference writes into these owners fail, including writes into Copied fields.
- Copied records and their ordinary holders use normal Go operations. Pointers
  to ordinary Copied storage are allowed; pointers do not satisfy the Copied proof.
- Each record in a proved field needs its own marker. Unmarked structs can
  hold Copied values, [Frozen handles](common-forms.md#ordinary-frozen-handle-slots--exhaustive),
  and [Frozen pointers](frozen-records.md#stored-pointers--exhaustive),
  including their ordinary containers. They cannot own Frozen records by value
  or Local data. See the complete [Copied forms](copied.md).
- Native containers, unmarked interfaces, functions, and Local values are not Frozen fields.
- A Cell is mutable storage, not a Frozen value or read result.

Embed `im.FrozenInterface` in a named interface to permit proved Frozen record
implementations. It is a Frozen type, not a read result. Use its generated local
projection when reading through a Local value. See the complete
[interface forms](frozen-interfaces.md).

## API requirements — exhaustive

| API | Type requirement |
| --- | --- |
| `Cell[T]`, `Projection[T]`, `ProjectionOf(value)` | T is a Frozen type. |
| `FrozenSlice[E]`, `FrozenMap[K,V]` | Each argument is a Frozen type; keys are also Go-comparable. |
| `ProjectString(value)` | A native string type. |
| `frozen.Projection()` | FrozenString, FrozenSlice[E], or FrozenMap[K,V]. |
| [Generic projection adapters](projection-primitives.md#projection-operations--exhaustive) | A Projection of the matching string, FrozenString, FrozenSlice, or FrozenMap type. |
| `IsNil(Projection[*T])` | T is a Frozen record. |
| `Value(Projection[T])` | T is both Frozen and a read result: Copied, UncheckedValue, or an unnamed array of these values. |
| `CopyRecord(Projection[T])` | T is a concrete Frozen record; returns a projection of a new copied root. |
| `Cell[T].LoadPointer()`, `SliceProjection[T].AtPointer(i)` | T is a concrete Frozen record; the result is Projection[*T]. |
| `Projection[T].Unsafe()` | T uses the same Frozen proof; the result is T with its shared storage. |
| `SliceProjection[E].Clone()` | E is Copied. |
| `MapProjection[K,V].Clone()` | K and V are Copied. |
| Exported service method data result | Read result; the [result signatures](service-results.md) also permit an error. |

The service rule does not change a struct's field rules. A Frozen record can be
stored in a Cell or read through Projection. For data, a service returns its generated
projection, an im.Local result, an unchecked handle, or a separate im.Copied value. A Frozen record
with only numeric fields still keeps the Frozen rule.

## Rejected examples — non-exhaustive

| Rejected | Use instead |
| --- | --- |
| `struct { Mark im.Frozen }` or `struct { *im.Frozen }` | Embed `im.Frozen` by value. |
| `struct { im.Frozen; Count }` | Use a named data field: `struct { im.Frozen; Count Count }`. |
| `im.Projection[im.Frozen]` | Embed the marker in a record and use that record. |
| `struct { im.Frozen; Text im.StringProjection }` | Use `im.Local` for a local result. |
| `struct { im.Local; Text string }` | Use `im.StringProjection`. |
| `struct { im.Local; Text im.FrozenString }` | Use `im.StringProjection`. |
| `struct { im.Local; Count FrozenCount }`, where FrozenCount embeds im.Frozen | Use `FrozenCountProjection`, or read its numeric fields into a Local record. |
| `type Counts [2]int` as protected storage | `type Counts struct { im.Frozen; Values [2]int }`. |
| `im.Value(textProjection)` with a string or FrozenString payload | Use the matching [projection adapter](projection-primitives.md#projection-operations--exhaustive), then Unsafe for shared access or Clone for independent storage. |
| `im.Value(countProjection)` with `countProjection im.Projection[FrozenCount]` | Use a generated projection, or copy individual numeric fields. |
| Save a Local record in a package variable | Use im.Copied when every field meets its value-only rule, or copy the required data into independent storage. |
