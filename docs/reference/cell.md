# Cells

A generated SettingsCell publishes Frozen Settings values through `im.Cell[Settings]`.
Its Load method returns SettingsProjection. Use the [generation requirements](rules.md#required-generation)
and [Frozen type rule](types.md) for the payload.

## Generated Cell methods — exhaustive

| Method | Result |
| --- | --- |
| `Store(value)` | Copy T into a new root and publish it atomically. |
| `Load()` | Return a local typed projection of the selected record without a record copy. |
| `LoadShared()` | Copy the selected T with permission to keep its shared storage; return zero T for an empty Cell. |

A zero generated Cell returns a nil projection. Use IsNil before reading a projection that
can be empty. Each load selects one version. Separate loads can select different
versions.

Code with the Cell can use LoadShared when it needs a persistent Frozen value. A Projection
has no LoadShared or Unwrap operation. Service data results require a read result, not a Cell.

## Cell storage

An `im.Cell[T]` is a raw slot. A holder is an unmarked struct with a named slot
or another named holder field. Generated Cell types are holders. A holder is
not an `im.Local` record. The slot stores Frozen data; Load returns a Local
projection. Service receivers keep their [service rules](services.md).

| Operation | Holder pointer `*Cache` | Holder value `Cache` | Raw `im.Cell[T]` | Local load result |
| --- | --- | --- | --- | --- |
| Copy, store, return, or pass | Ordinary Go rules | Rejected | Rejected | [Local forms](local.md) only |
| Use in an interface, callback, closure, or generic type argument | Ordinary Go rules | No value copies | No value copies | Rejected |
| Change storage | Change ordinary fields or rebind the pointer | No replacement; change ordinary fields | Store only | No field writes |
| Take an address | Ordinary pointer indirection | Address the holder without a copy | Rejected | [Local pointer forms](local.md#pointer-forms) only |

A holder pointer shares one slot. Copying the pointer does not copy the slot.
The holder, its nested holder fields, and its raw slots cannot be copied or
replaced. Frozen fields, Local values, and projection results keep their type
and source rules.

## Slot and holder value forms — exhaustive

| Form | Example |
| --- | --- |
| Package raw slot without an initializer | `var raw im.Cell[Settings]` |
| Named slot or holder field in an unmarked struct | `type Cache struct { settings SettingsCell; Calls int }`, `type Slot struct { value im.Cell[Settings] }` |
| Local or package holder variable without an initializer | `var cache Cache` |
| Construct a holder through its address; leave slot and holder fields zero | `cache := &Cache{Calls: 1}`, `cache := new(Cache)` |
| Take the address of a holder variable or named holder field | `p := &cache`, `p := &outer.cache` |
| Select a field without copying its holder | `cache.Calls++`, `n := (*p).Calls`, `p := &cache.Calls` |
| Select a pointer-receiver method, or a promoted method on an ordinary field, without copying its holder | `cache.Read()`, `read := cache.Count`, `outer.cache.Read()` |
| Use a slot or holder variable inside a closure | `func() { v := raw.Load(); _ = v }`, `func() { cache.Calls++ }` |
| Store through a direct raw slot in its own synchronous statement | `raw.Store(next)`, `slot.value.Store(next)` |
| Read through a direct raw slot in a common statement form | `v := raw.Load()`, `return slot.value.Load()`, `saved := raw.LoadShared()` |
| Alias, package-qualified slot, or parentheses | `type Current = SettingsCell`, `v := config.Current.Load()`, `v := (raw).Load()` |

Raw slots cannot be copied, addressed, returned, passed to helpers, or embedded.
Holder values cannot be copied, replaced, embedded, or put in arrays or native
containers. Calls that carry protected values use the
[common forms](common-forms.md). Unlisted slot and holder value operations are
rejected. A closure refers to the same holder variable; a value copy such as
`copy := cache` is still rejected inside that closure. Load projections inside
a closure; do not capture projections loaded outside it.

## Holder pointer forms

Holder pointers and ordinary structs that contain only those pointers use Go
rules. The following list covers their permitted operation groups. Operations
on the pointed-to slot still need a form from the exhaustive list above.

| Form | Example |
| --- | --- |
| Construct, copy, replace, or return a pointer | `func New() *Cache { return new(Cache) }`, `other := cache`, `cache = other` |
| Initialize ordinary holder fields | `cache := &Cache{Calls: 1}` |
| Store in a package variable, field, array, slice, map, or channel | `var saved *Cache`, `type Parent struct { Child *Cache }`, `values := []*Cache{cache}` |
| Copy an ordinary wrapper that contains a holder pointer | `other := Parent{Child: cache}` |
| Use aliases, defined pointer types, conversions, or nil | `type Pointer *Cache`, `p := Pointer(cache)`, `p = nil` |
| Address or rebind a pointer | `p := &cache`, `*p = new(Cache)` |
| Use multiple bindings, tuple results, comparisons, switches, range, or select | `a, b = b, a`, `if cache != nil {}`, `for _, p := range values { p.Count() }` |
| Use fixed or variadic arguments and results | `use(cache)`, `useAll(cache, other)`, `return cache, nil` |
| Use interfaces and type assertions | `var value any = cache`, `cache = value.(*Cache)` |
| Use function values, closures, go, or defer with ordinary arguments and results | `f := func() *Cache { return cache }`, `go use(cache)`, `defer use(cache)` |
| Use pointer methods, method values, or method expressions with ordinary arguments and results | `cache.Count()`, `count := cache.Count`, `(*Cache).Count(cache)` |
| Embed a holder pointer and promote its pointer methods | `type Parent struct { *Cache }`, `parent.Count()` |
| Use a pointer as a generic type argument or constraint | `identity(cache)` for `func identity[T any](v T) T`, `type Set interface { ~*Cache }` |

Methods that pass or return protected values still need the common forms. For
example, `view := cache.settings.Load()` is allowed, but saving its Load method
as a callback is rejected because its result is Local.

A generic type argument of `Cache` is rejected, including for a parameter
`p *T`: the generic body could copy `*p`. Use a type argument of `*Cache` and a
parameter `p T`. Generic holder declarations that store slots by value remain
rejected. The [package boundary](rules.md#package-boundary) still requires a Go
body for a function whose contract contains a Cell.

## Raw Cell methods — exhaustive

| Method | Result |
| --- | --- |
| `Store(value)` | Publish a private T root. |
| `Load()` | Copy T into Projection[T]; read zero T if the slot is empty. |
| `LoadPointer()` | Return Projection[*T] without copying T; T must be a concrete Frozen record. An empty slot returns a nil pointer projection. |
| `LoadShared()` | Copy T for persistent use; read zero T if the slot is empty. |

Generated Cell.Load uses the raw LoadPointer method.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `copy := raw` or `p := &raw` | Call the slot directly. |
| `var current im.Cell[Settings]` inside a function | Declare a package slot or a named Cell field. |
| `copy := *cache` or `*cache = Cache{}` | Copy or replace the pointer: `other := cache`, `cache = new(Cache)`. |
| `p := &slot.value` for a raw slot | Call `slot.value.Load()` directly. |
| A holder value receiver | Declare a pointer receiver. |
| `parent.value.Load()` through an embedded holder pointer | Select the embedded pointer explicitly: `parent.Cache.value.Load()`. |
| `read := settings.Load` | `projection := settings.Load()` |
| `go raw.Store(next)` or `defer cache.settings.Load()` | Use a function that calls Store or Load synchronously. |
| `im.Cell[LocalResult]` | Store Frozen data; build local projections when reading. |
| Return `settings.LoadShared()` from a service | Return `settings.Load()`. |

Store allocates one root for T with nonzero size. Generated Load performs one
atomic load without a record copy or allocation. LoadShared copies T after one
atomic load; it does not copy collection or string backing storage.
