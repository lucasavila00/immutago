# Frozen records

Embed `im.Frozen` in immutable stored data. Its package must use the
[required generation step](rules.md#required-generation):

```go
type Settings struct {
    im.Frozen
    Title string
    Limit int
}

settings := Settings{Title: "standard", Limit: 100}
saved := settings
```

Both values share the string bytes. Frozen means immutable, not a deep copy.
The [type rules](types.md) define allowed fields. Use a
[Frozen interface](frozen-interfaces.md) for a polymorphic field.

| Field declarations — exhaustive | Example |
| --- | --- |
| Embed exactly one Frozen marker by value; aliases work | `im.Frozen` or `type Marker = im.Frozen; type Settings struct { Marker; Limit int }` |
| Give each data field a name and a Frozen type | `Count Count`, `Child Child`, `Next *Settings` |

Other field declarations are rejected. Data fields cannot be embedded. A named
field prevents Go from promoting its methods to the Frozen record. Such a
promoted method can take an implicit field address, including through an interface.

| Allowed forms — exhaustive | Example |
| --- | --- |
| Construct, copy, and read | `Settings{Limit: 100}`, `saved := settings`, `n := settings.Limit` |
| Store in Frozen fields or shared frozen collections | `im.FrozenSlice[Settings]` |
| Store or return a fresh Frozen pointer | `return &Settings{Limit: 100}` |
| Publish through a Cell | `cell.Store(settings)` |
| Read through a generated Cell | `projection := cell.Load()` |
| Keep the selected root without a record copy | `saved := cell.Load().Unsafe()` |

Use the [common source forms](common-forms.md) for statements, calls, methods,
and pointer construction. The [projection generator](projections.md) generates
SettingsProjection and SettingsCell from this declaration.

## Stored pointers — exhaustive

A `*Settings` points to an immutable object. The pointer slot can change without
changing that object. An ordinary request struct can keep one selected version:

```go
type requestState struct {
    settings *Settings
    calls int
}

func (s *requestState) begin(cell *SettingsCell) {
    s.settings = cell.Load().Unsafe()
    s.calls++
}
```

| Storage or use | Complete rule |
| --- | --- |
| Frozen pointer handle or ordinary carrier | All Go storage and transport forms are allowed. The examples below include pointer slots, methods, interfaces, generics, and channels. |
| Selected or dereferenced Frozen object | Use the [common forms](common-forms.md). Reads and value copies are allowed; target writes and interior addresses are rejected. |
| Native slice or map passed to Freeze | Use an exclusive [builder](frozen.md#build-before-freezing). Freeze consumes it. |
| Exported service method result | Use the [service result signatures](service-results.md). A Frozen pointer is not a read result. |

| Ordinary Go examples | Source |
| --- | --- |
| Store in an ordinary struct or nested carrier | `type State struct { Settings *Settings; Next *State }` |
| Store in native containers | `[2]*Settings{p, nil}`, `[]*Settings{p}`, `map[string]*Settings{"current": p}`, `make(chan *Settings)` |
| Replace a field, element, or pointer slot | `s.Settings = p`, `values[i] = nil`, `slot := &s.Settings; *slot = p` |
| Read the selected object | `n := s.Settings.Limit`, `n := versions[key].Limit` |
| Bind or return several pointers | `p, ok := versions[key]`, `p, q := pair()` |
| Box, assert, or pass through a pool | `var value any = p`, `value.(*Settings)`, `pool.Put(state)` |
| Use generics, callbacks, or closures | `identity(p)`, `callback(p)`, `func() { use(p) }`, `func read[T ~*Settings](p T) int { return (*p).Limit }` |
| Use methods, method values, and method expressions | `p.Read()`, `read := p.Read`, `(*Settings).Read(p)` |
| Use go, defer, channels, or select | `go use(p)`, `defer use(p)`, `ready <- p` |
| Use aliases or defined pointer types | `type Pointer = *Settings`, `type Selected *Settings` |

The checker rejects statically typed writes to the Frozen target, including
after a type assertion, a generic call, or a promoted field selection. Ordinary Go
API contracts still apply: do not pass the pointer to an API that changes its
target, such as `json.Unmarshal(data, p)`. The checker does not prove mutation
effects through ordinary interfaces, reflection, unsafe code, or dependencies.
A defined pointer such as `type Selected *Settings` can use the forms above,
but cannot be a Frozen field, Cell payload, or frozen collection argument.
Use `type Selected = *Settings` for those [Frozen type requirements](types.md).

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `type Settings struct { im.Frozen; Count }` | `type Settings struct { im.Frozen; Count Count }`. Copy `settings.Count` into an ordinary variable before changing it. |
| `type Settings struct { im.Frozen; *Child }` | `type Settings struct { im.Frozen; Child *Child }`. |
| `settings.Limit = 200` | Construct another Settings value. |
| `p := &settings` | `p := new(settings)`; copy into fresh storage. |
| `s.Settings.Limit = 200` or `*s.Settings = Settings{}` | Replace the slot: `s.Settings = &Settings{Limit: 200}`. |
| `p := &s.Settings.Limit` | Copy the scalar: `n := s.Settings.Limit`. |
| `type State struct { Settings Settings }` | Store `*Settings`, or make State a Frozen record. |
| Return Settings from a service, including a record with only numeric fields | Return SettingsProjection or an im.Local result. |
