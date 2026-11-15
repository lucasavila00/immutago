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
| `LoadSharedPointer()` | Return the selected *T for persistent use without copying T; return nil for an empty Cell. |

A zero generated Cell returns a nil projection. Use IsNil before reading a projection that
can be empty. Each load selects one version. Separate loads can select different
versions.

Code with the Cell can use LoadShared when it needs a persistent Frozen value. A Projection
has no LoadShared or Unwrap operation. Service data results require a read result, not a Cell.
Use LoadSharedPointer to keep one version in an ordinary request struct or pooled object.
The [stored pointer forms](frozen-records.md#stored-pointers--exhaustive) permit slot
replacement and keep the selected Frozen object immutable. Earlier pointers remain
valid after Store publishes a new root.

## Cell and holder forms — exhaustive

An `im.Cell[T]` is a raw slot. A holder is an unmarked struct with a named slot
or another named holder field. Generated cell types are holders and use these
same rules. Service receivers keep their ordinary [service rules](services.md).

| Form | Example |
| --- | --- |
| Package raw slot without an initializer | `var raw im.Cell[Settings]` |
| Named slot or holder field in an unmarked struct | `type Cache struct { settings SettingsCell }`, `type Slot struct { value im.Cell[Settings] }` |
| Local or package holder variable without an initializer | `var cache Cache` |
| New holder pointer in a local variable | `cache := &Cache{}`, `cache := new(Cache)` |
| Copy a holder pointer to a local variable | `other := cache` |
| Pass a holder pointer to a named, non-generic function or concrete method with an exact fixed parameter type | `publish(cache)` with `func publish(c *Cache)`; `state.begin(cache)` with `func (s *State) begin(c *Cache)` |
| Capture a Cell slot or holder pointer | `func() { v := raw.Load(); _ = v }` |
| Call a named holder function in a goroutine | `go publish(cache)` with `func publish(c *Cache)` |
| Declare a direct, non-generic method on a holder pointer | `func (c *Cache) Read() SettingsProjection { return c.settings.Load() }` |
| Call that method through its exact pointer type | `cache.Read()` for `cache *Cache` |
| Call that method through an addressable holder variable or direct field | `settings.Load()`, `cache.settings.Load()` |
| Store through a direct raw slot in its own synchronous statement | `raw.Store(next)`, `slot.value.Store(next)` |
| Read through a direct raw slot in a common statement form | `v := raw.Load()`, `return slot.value.Load()`, `saved := raw.LoadShared()` |
| Load a persistent Frozen pointer from a direct raw slot | `saved := raw.LoadSharedPointer()`, `state.settings = raw.LoadSharedPointer()` |
| Alias, package-qualified slot, or parentheses | `type Current = SettingsCell`, `v := config.Current.Load()`, `v := (raw).Load()` |

Raw slots cannot be copied, addressed, returned, passed to helpers, or embedded.
A holder pointer cannot enter a field, container, or package variable, or be
returned. Holder values cannot be copied, replaced, embedded, or put in arrays
or native containers. Calls use the
[common forms](common-forms.md); unlisted Cell operations are rejected.
Load projections inside a closure; do not capture projections loaded outside it.

## Raw Cell methods — exhaustive

| Method | Result |
| --- | --- |
| `Store(value)` | Publish a private T root. |
| `Load()` | Copy T into Projection[T]; read zero T if the slot is empty. |
| `LoadPointer()` | Return Projection[*T] without copying T; T must be a concrete Frozen record. An empty slot returns a nil pointer projection. |
| `LoadShared()` | Copy T for persistent use; read zero T if the slot is empty. |
| `LoadSharedPointer()` | Return the current *T without a copy; T must be a concrete Frozen record. An empty slot returns nil. |

Generated Cell.Load uses the raw LoadPointer method.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `copy := raw` or `p := &raw` | Call the slot directly. |
| `var current im.Cell[Settings]` inside a function | Declare a package slot or a named Cell field. |
| `copy := *cache` or `return cache` | Keep the holder pointer local or pass it to a named function. |
| `p := &cache.settings` | Call `cache.settings.Load()` directly. |
| A value receiver, promoted call, method value, or method expression on a holder | Declare a direct pointer-receiver method and call it directly. |
| `read := settings.Load` | `projection := settings.Load()` |
| `go settings.Store(next)` or `defer cache.Read()` | Call synchronously; use `go publish(cache)` for a named publisher. |
| `im.Cell[LocalResult]` | Store Frozen data; build local projections when reading. |
| Return `settings.LoadShared()` from a service | Return `settings.Load()`. |
| Return `settings.LoadSharedPointer()` from a service | Return `settings.Load()`. |
| `raw.LoadSharedPointer()` with `raw im.Cell[int]` | Use `raw.LoadShared()` for the scalar value. |

Store allocates one root for T with nonzero size. Generated Load performs one
atomic load without a record copy or allocation. LoadShared copies T after one
atomic load; it does not copy collection or string backing storage.
LoadSharedPointer performs one atomic load without a record copy or allocation.
