# Snapshots and local reads

| API | Purpose |
| --- | --- |
| `im.Snapshot` | Mark immutable stored data. |
| `im.Local` | Mark an immutable record that can contain local read views. |
| `im.Cell[T]` | Publish a Snapshot value atomically. |
| `im.View[T]` | Read a Snapshot without exposing its shared storage. |

## Type rules — exhaustive

Each rule applies recursively. Aliases keep the underlying rule.

| Category | Types |
| --- | --- |
| Snapshot | Booleans, numbers, strings, unnamed arrays of Snapshots, structs marked Snapshot, pointers to Snapshot structs, frozen slices/maps of Snapshots, trusted UncheckedValue handles, and service references. |
| Copied | Booleans, numbers, opaque service references, and unnamed arrays or Snapshot records whose fields are all Copied. |
| Local | View, StringView, SliceView, MapView, Local records, and unnamed arrays of Local or Copied elements. |
| Read result | Copied or Local. |

- Embed one marker by value. Named marker fields, pointer markers, and two markers fail.
- Snapshot fields must be Snapshots. They cannot contain Local values.
- Local fields must be read results. They cannot contain shared data handles.
- Set marked fields at construction. Later field and element writes fail.
- Native slices, maps, channels, interfaces, and function values are not Snapshot fields.
- UncheckedValue keeps its explicit trust contract. A view cannot unwrap it.

## Storage and calls — exhaustive

| Operation | Snapshot value | Local value |
| --- | --- | --- |
| Local variable or direct concrete call | Allowed | Allowed |
| Return from an ordinary function | Allowed | Allowed; restrictions follow the result |
| Package variable, Snapshot field, Cell, or shared frozen collection | Allowed | Rejected, including through another record or array |
| Exported service result | Copied values only | Allowed |
| Interface, callback, closure capture, user generic, go, or defer | Use the common protected-source rules | Rejected |

Services accept no result or one read result. Init, Run, and runtime methods
keep their existing contracts. There is no service-only field proof:
Cell[T] and View[T] use the same Snapshot proof; result checks use the general
Copied and Local categories.

## Cells

```go
type Settings struct {
    im.Snapshot
    Title string
    Limit int
}

var current im.Cell[Settings]

func publish() {
    current.Store(Settings{Title: "standard", Limit: 100})
}

func read() im.View[Settings] { return current.Load() }
```

Store copies T into a new immutable root and publishes it. Load returns a local
view of the selected value. A zero cell reads zero T.

Shared returns T for code that holds the cell and needs persistent access.
A View has no Shared or Unwrap operation. A cell is a mutable capability;
it cannot be a Snapshot field or service result.

## Select and copy

```go
view := current.Load()
limit := im.Copy(im.Select(view, func(s Settings) int { return s.Limit }))
title := im.Text(im.Select(view, func(s Settings) string { return s.Title }))
println(limit, title.Len())
saved := title.Clone()
```

Select takes an inline function literal with one parameter and one return.
Its return must be a path from that parameter: fields, pointer dereferences,
and fixed-array indices. An index is an integer constant or an ordinary integer
local. Calls, writes, slicing, conversions, addresses, and other captures fail.
Select always returns another View; it never returns the selected field raw.

Copy requires Copied T. Text accepts a View of a string type and returns a
StringView. A pointer view supports IsNil before a selector dereferences it.

## Collection reads — exhaustive

FreezeSlice and FreezeMap retain their exclusive-builder checks. ViewOf wraps
an existing Snapshot value. Slice and Map turn views of frozen collections
into their corresponding local read APIs.

| Operation | StringView | SliceView[E] | MapView[K,V] |
| --- | --- | --- | --- |
| Len | Byte count | Element count | Entry count |
| At | Byte | View[E] | — |
| Get | — | — | View[V], bool; accepts K for lookup |
| All | — | Index and View[E] | View[K] and View[V] |
| Clone | Independent native string | Native slice; E must be Copied | Native map; K and V must be Copied |

All is consumed by direct range. Map keys always remain views, including string
keys. For strings or nested records, select and copy each required value; there
is no hidden recursive Clone. No view exposes a substring or writable storage.

## Costs

Freeze and view creation do not copy backing storage. View[T] copies the Go
value T, such as a string/slice/map handle or fixed-size record. Select can copy
its selected value. Cell Store allocates one root; Load performs one atomic load
and a T value copy. String Clone copies bytes; collection Clone copies entries.

Local views can keep storage alive while their function runs. They cannot be
saved in persistent storage. Copies made with Clone have independent storage.
