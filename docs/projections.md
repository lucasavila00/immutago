# Typed projections

Every package with user-declared `im.Frozen` or `im.Local` records requires
generation. Frozen records get typed projections; Local records need no additional
wrapper. Field methods return Copied values or local projections.

## Generate projections

Add the generator with the [install command](getting-started.md#install).
Declare the record and the generation command in its package:

```go
package settings

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import "github.com/lucasavila00/immutago/pkg/im"

type Settings struct {
    im.Frozen
    Title im.FrozenString
    Limit int
}
```

Run `go generate ./...`. The command writes `zz_immutago.gen.go` in each package
with this directive. It uses named package-scope records from non-test Go files.
It generates projections for each concrete Frozen record and records the Local declarations.
Run it again after a field changes. Keep this output with the source. The checker
rejects a missing directive, missing output, or stale output. Generated-name
conflicts are errors.

Load gives a SettingsProjection. Read its fields through generated methods:

```go
var current SettingsCell

func publish() {
    current.Store(Settings{Title: im.FreezeString("standard"), Limit: 100})
}

func read() SettingsProjection { return current.Load() }

func use() {
    projection := read()
    limit := projection.Limit()
    title := projection.Title()
    println(limit, title.Len(), title.EqualString("standard"))
    println(title.Unsafe())
}
```

Equal and EqualString compare text without allocating. Unsafe supplies a shared
native string to ordinary Go APIs; Clone gives independent storage. The integer
result is Copied data and can be stored directly. An empty Cell or zero
SettingsProjection has IsNil true. Each Load selects one version for all its field reads.

## Operations on record values — exhaustive

| Form | Result |
| --- | --- |
| `var current SettingsCell` | Empty publication slot. |
| `current.Store(value)` | Publish a Settings value. |
| `current.Load()` | Read the selected record through SettingsProjection without copying the record. |
| `current.Shared()` | Read the stored Settings with its shared storage. |
| `var projection SettingsProjection` or `SettingsProjection{}` | A nil projection. |
| `projection.IsNil()` | Report whether the projection has a record. |
| `projection.Field()` | Read the declared field through its generated result type. |

## Field results — exhaustive

Each direct field gets a method with the same name, except marker fields and
blank fields. An unexported field has an unexported method. Field names Local,
IsNil, and immutagoProjection conflict with generated members and are rejected.

| Field type | Generated result |
| --- | --- |
| Copied scalar, marker, or service reference | The field's own type, such as `int` or `supervise.Ref[Worker]`. |
| Native string or FrozenString | `im.StringProjection`. |
| FrozenSlice[E] | `im.SliceProjection[E]`. |
| FrozenMap[K,V] | `im.MapProjection[K,V]`. |
| Frozen record or pointer to one | The record's generated typed projection. |
| Unnamed array | A typed projection with Len and At; At uses this same result table. |
| UncheckedValue[T] | Opaque `im.Projection[im.UncheckedValue[T]]`. |

Record and array fields return projections even when their types are Copied. A
field read does not copy the containing record or array. Reading a field through
a nil projection panics. Array At panics if the index is out of range.

Cell placement follows the [Cell rules](cell.md). Generated code uses the
[projection primitives](projection-primitives.md) and direct field reads.
The checker verifies the exact generated output before permitting its internal
Read operation. It still checks all generated types, writes, captures, calls,
and value transport.

## Collection elements

A slice of records supports pointer reads into its immutable storage. For
`items im.SliceProjection[Settings]`:

```go
for index := 0; index < items.Len(); index++ {
    item := NewSettingsProjection(items.AtPointer(index))
    println(item.Title().Unsafe())
}
```

| Constructor for a record value — exhaustive | Input and result |
| --- | --- |
| `NewSettingsProjection(value)` | Takes `im.Projection[*Settings]` and returns SettingsProjection without copying the record. |

For an unexported record such as state, the constructor is `newStateProjection`.

A map with `*Settings` values already returns `im.Projection[*Settings]` from Get.
For a map with Settings values, request the record copy explicitly:

```go
value, found := entries.Get(key)
if found {
    item := NewSettingsProjection(im.CopyProjection(value))
    println(item.Title().Unsafe())
}
```

CopyProjection copies one record into a new root. It does not copy the strings or
collections that the record refers to. Generated projections do not copy map values
into roots automatically.

## Local transport — exhaustive

| Form | Example |
| --- | --- |
| Local declaration, copy, replacement, or discard | `v := current.Load()`, `other := v`, `_ = other` |
| Field of a Local record | `type Reply struct { im.Local; Settings SettingsProjection }` |
| Local unnamed array | `projections := [2]SettingsProjection{v}` |
| Exclusive local native builder | `values := []SettingsProjection{v}`; use the [Local builder forms](local.md). |
| Direct concrete argument or result | `func read(v SettingsProjection) int`, `return v` |

These rules follow nested fields and arrays. Projections cannot enter package
variables, Frozen fields, Cells, frozen entries, service start arguments,
interfaces, callbacks, closures, user generics, go, or defer.
The lists are complete; unlisted operations are rejected.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| Save `current.Load()` in a package variable | Keep a copied field or explicit Clone result. |
| `fmt.Sprintf("%s", projection.Title())` | `fmt.Sprintf("%s", projection.Title().Unsafe())`; Clone instead when independent storage is needed. |
| Capture `projection` in a closure | Read Copied fields or make independent copies before the closure. |
| `go useProjection(projection)` or `defer useProjection(projection)` | Make the direct synchronous call. |
| Read a raw pointer or UncheckedValue from a projection | Keep that operation in code with persistent Frozen access. |
| Declare `SettingsProjection` or `SettingsCell` before generation | Use another source name; these names belong to the generated API. |
| A field name that conflicts with a generated member, such as Local | Rename the source field. |
| A function-local, anonymous, or test-only marked record | Declare a named type at package scope in a non-test Go file. |

Generated projections keep a pointer to the selected immutable record. Primitive
Projection[T] values and explicit copies have their ordinary Go value-copy costs.
