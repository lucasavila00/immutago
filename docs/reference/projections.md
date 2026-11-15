# Generated projections

The [required generation step](rules.md#required-generation) creates typed
projections for Frozen records and Frozen interfaces. Record getters return
Copied values, unchecked handles, or further Local projections. Local and Copied records need no
generation. See [Get started](../guides/getting-started.md) for a first program.

## Generated names and operations — exhaustive

For a Frozen record named Settings:

| Form | Input or result |
| --- | --- |
| `SettingsProjection` | Local projection of a Settings record. |
| `NewSettingsProjection(value)` | Takes `im.Projection[*Settings]`; returns SettingsProjection without copying the record. |
| `var p SettingsProjection` or `SettingsProjection{}` | Nil projection. |
| `p.IsNil()` | Report whether the projection has a record. |
| `p.Unsafe()` | Return the selected *Settings without a record copy or allocation; nil for an empty projection. |
| `p.Field()` | Read a declared field using the result table below. |
| `SettingsCell` | Publication slot with Store and Load; use the [complete Cell forms](cell.md). |

For an unexported record such as state, the constructor is `newStateProjection`.
Generated-name conflicts are errors. Do not declare the generated types yourself.
Use the complete [Local transport forms](local.md#value-forms--exhaustive) and
[common source forms](common-forms.md). Unlisted protected operations are rejected.

## Field results — exhaustive

Each direct field gets a method with the same name, except marker and blank
fields. An unexported field has an unexported method. Field names Local, IsNil, Unsafe,
and immutagoProjection conflict with generated members and are rejected.

| Field type | Generated result |
| --- | --- |
| Copied scalar, Copied record, or service reference | The field's own type: `int`, `Score`, or `supervise.Ref[Worker]`. |
| Native string or FrozenString | `im.StringProjection`. |
| FrozenSlice[E] | `im.SliceProjection[E]`. |
| FrozenMap[K,V] | `im.MapProjection[K,V]`. |
| Frozen record or pointer to one | The record's generated typed projection. |
| Frozen interface | The interface's generated local method projection. |
| Unnamed array | A typed projection with Len and At; At uses this result table. |
| UncheckedValue[T] | The handle itself: `im.UncheckedValue[T]`. |

Arrays return projections, including arrays of Copied values. A Copied record
field returns a complete value copy. A field read does not copy its containing
record or array. Reading a field through a nil record projection panics. Array At panics
if the index is out of range.

Unsafe keeps the exact record selected by the projection. For an inline child,
`p.Child().Unsafe()` points into its parent; keeping that pointer can keep the
parent alive. For a pointer field, it returns that field's pointer. Neither form
copies a record. The caller controls retention and must not change the target
through reflection, unsafe code, or another API. Typed writes remain rejected.
Use the [stored pointer forms](frozen-records.md#stored-pointers--exhaustive).

## Collection elements — exhaustive

| Element | Read through a generated projection |
| --- | --- |
| Frozen record in a slice | `item := NewSettingsProjection(items.AtPointer(i))` |
| Frozen pointer in a slice | `item := NewSettingsProjection(items.At(i))` |
| Frozen pointer in a map | `value, found := entries.Get(key)`; when found, `item := NewSettingsProjection(value)` |
| Frozen record value in a map | `value, found := entries.Get(key)`; when found, `item := NewSettingsProjection(im.CopyRecord(value))` |
| Frozen interface in a slice or map | `local := NewScorerProjection(items.At(i))`, or pass the projection from map Get; use the [interface forms](frozen-interfaces.md). |

CopyRecord copies one record into a new root. It shares the immutable strings
and collections referenced by that record. Generated projections never make
this map-value copy automatically. See [operation costs](performance.md).

## Rejected examples — non-exhaustive

| Rejected | Allowed replacement |
| --- | --- |
| `im.Value(raw)` with `raw im.Projection[Settings]` | Use a generated projection to read the required fields. |
| `NewSettingsProjection(items.At(i))` for a slice of Settings values | `NewSettingsProjection(items.AtPointer(i))` |
| Change `p.Unsafe().Limit` | Construct another Settings value. |
| Declare SettingsProjection or SettingsCell before generation | Use another source name. |
| A source field named Local | Rename the field. |

Only verified generated code can form interior Frozen-record addresses.
The public Unsafe operation returns the selected value without another Cell load. It still receives all normal type, write,
capture, call, and transport checks. The [architecture](../arch/ownership.md#generated-projections)
describes those checks.
