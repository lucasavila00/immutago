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

| Allowed forms — exhaustive | Example |
| --- | --- |
| Construct, copy, and read | `Settings{Limit: 100}`, `saved := settings`, `n := settings.Limit` |
| Store in Frozen fields or shared frozen collections | `im.FrozenSlice[Settings]` |
| Store or return a fresh Frozen pointer | `return &Settings{Limit: 100}` |
| Publish through a Cell | `cell.Store(settings)` |
| Read through a generated Cell | `projection := cell.Load()` |

Use the [common source forms](common-forms.md) for statements, calls, methods,
and pointer construction. The [projection generator](projections.md) generates
SettingsProjection and SettingsCell from this declaration.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `settings.Limit = 200` | Construct another Settings value. |
| `p := &settings` | `p := new(settings)`; copy into fresh storage. |
| Return Settings from a service, including a record with only numeric fields | Return SettingsProjection or an im.Local result. |
