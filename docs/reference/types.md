# Type rules

These categories define fields, Cells, projections, and service data results.

| Category — exhaustive | Types |
| --- | --- |
| Frozen | Booleans, numbers, native strings, FrozenString, unnamed arrays of Frozen values, Frozen and Copied records, unnamed pointers to Frozen records (and aliases of those pointers), [Frozen interfaces](frozen-interfaces.md), FrozenSlice/FrozenMap with Frozen arguments, UncheckedValue. |
| Copied | Booleans, numbers, Copied records, and unnamed arrays of Copied values. |
| Local | Projection, StringProjection, SliceProjection, MapProjection, Local records, and unnamed arrays whose elements are Local. |
| Read result | The types in the [read-result list](#read-results--exhaustive) below. |

Defined scalar types and aliases keep their category. Named arrays need a
Frozen, Local, or Copied record wrapper. Copied excludes native strings and
pointers. Its structs require `im.Copied`.
Frozen records and UncheckedValue are not Copied.
Defined pointer types such as `type Selected *Settings` can be stored under the
[ordinary pointer rules](frozen-records.md#stored-pointers--exhaustive), but are
not Frozen payload types. Use `type Selected = *Settings` for a pointer alias.
Embed a marker to declare a record category. The markers themselves are not
Frozen, Copied, or read-result payload types.

## Read results — exhaustive

| Type | Example |
| --- | --- |
| Copied value | `int`, `Score` with `im.Copied` |
| Local value | `SettingsProjection`, `im.StringProjection`, `Reply` with `im.Local` |
| UncheckedValue | `im.UncheckedValue[Model]` |
| Named Frozen record with one named UncheckedValue field and no other data fields | `type Counter struct { im.Frozen; state im.UncheckedValue[counter] }` |
| Unnamed pointer to that Frozen record, including aliases | `*Counter`, `type Pointer = *Counter` |
| Unnamed array of read results, recursively | `[2]Score`, `[2]*Counter`, `[2]im.StringProjection` |

Aliases keep these types. The [UncheckedValue rules](unchecked-values.md) apply
to its payload. Other Frozen records are not read results, even with only numeric
fields. Use a generated projection or a Copied or Local record for those results.
Go's predeclared `error` is permitted by the [service result rule](service-results.md),
but is not a read result or a field type in Frozen, Local, or Copied records.

## Markers — exhaustive

| Marker | Required fields | Record use |
| --- | --- | --- |
| `im.Frozen` | Frozen fields | Can be copied, stored, and returned from ordinary functions. |
| `im.Local` | Read-result fields | Can be copied locally, passed directly, and returned; use local variables, Local records, or Local builders. |
| `im.Copied` | Boolean/numeric fields, unnamed arrays, and other Copied records | Can be stored by value in ordinary holders and returned by services; ordinary copies can change independently. |

- Embed exactly one marker by value; aliases work.
- Frozen records embed only their marker. Give every data field a name:
  `Count Count`, not an embedded `Count`. Local and Copied field rules do not change.
- Frozen declarations follow the [generation requirements](rules.md#required-generation).
  Local and Copied records need no generation.
- Set Frozen and Local fields at construction. Later field, element, or
  dereference writes into these owners fail, including writes into Copied fields.
- Copied records and their ordinary holders use normal Go operations. Pointers
  to ordinary Copied storage are allowed; pointers are not Copied values.
- Each record in a protected field needs its own marker. Unmarked structs can
  hold Copied values, [frozen collections and interfaces](common-forms.md#ordinary-collection-and-interface-slots--exhaustive),
  and [Frozen pointers](frozen-records.md#stored-pointers--exhaustive),
  including their ordinary containers. They can also hold UncheckedValue and
  the Frozen records in the read-result list. They cannot own other Frozen
  records by value or Local data.
- Native containers, unmarked interfaces, functions, and Local values are not Frozen fields.
- A Cell is mutable storage, not a Frozen value or read result.

Embed `im.FrozenInterface` in a named interface to permit Frozen record
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
| `Value(Projection[T])` | T is both Frozen and a read result. |
| `CopyRecord(Projection[T])` | T is a concrete Frozen record; returns a projection of a new copied root. |
| `Cell[T].LoadPointer()`, `SliceProjection[T].AtPointer(i)` | T is a concrete Frozen record; the result is Projection[*T]. |
| `Projection[T].Unchecked()` | T is Frozen; the result is T with its shared storage. |
| `SliceProjection[E].Clone()` | E is Copied. |
| `MapProjection[K,V].Clone()` | K and V are Copied. |
| Exported service method data result | Read result; the [result signatures](service-results.md) also permit an error. |

## Rejected examples — non-exhaustive

| Rejected | Use instead |
| --- | --- |
| `struct { Mark im.Frozen }` or `struct { *im.Frozen }` | Embed `im.Frozen` by value. |
| `struct { im.Frozen; Count }` | Use a named data field: `struct { im.Frozen; Count Count }`. |
| `im.Cell[Selected]` or a Frozen field of type Selected, with `type Selected *Settings` | Use `type Selected = *Settings`, where Settings is a Frozen record. |
| `im.Projection[im.Frozen]` | Embed the marker in a record and use that record. |
| `struct { im.Frozen; Text im.StringProjection }` | Use `im.Local` for a local result. |
| `struct { im.Local; Text string }` | Use `im.StringProjection`. |
| `struct { im.Local; Text im.FrozenString }` | Use `im.StringProjection`. |
| `struct { im.Local; Count FrozenCount }`, where FrozenCount embeds im.Frozen | Use `FrozenCountProjection`, or read its numeric fields into a Local record. |
| `type Counts [2]int` as protected storage | `type Counts struct { im.Frozen; Values [2]int }`. |
| `im.Value(textProjection)` with a string or FrozenString payload | Use the matching [projection adapter](projection-primitives.md#projection-operations--exhaustive), then Unchecked for shared access or Clone for independent storage. |
| `im.Value(countProjection)` with `countProjection im.Projection[FrozenCount]` | Use a generated projection, or copy individual numeric fields. |
| Save a Local record in a package variable | Use im.Copied when every field meets its value-only rule, or copy the required data into independent storage. |
