# Service results

Each exported service method returns no result or one [read result](types.md):
Copied data or a Local value. Init, Run, and runtime methods are exempt.
The rule also covers promoted methods.

| Allowed result — exhaustive | Example |
| --- | --- |
| No result | `func (c *Cache) Refresh()` |
| Copied scalar, marker, array, or Frozen record | `int`, `im.Frozen`, `[2]int`, `Usage` with only numeric fields |
| Service reference | `supervise.Ref[Worker]` |
| Local projection | Generated `SettingsProjection`, `im.StringProjection`, `im.MapProjection[string, int]` |
| Local marker, record, or unnamed array of read results | `im.Local`, `Prediction`, `[2]im.StringProjection` |

There is no separate field rule for services. Cell[T] and Projection[T] require the
same Frozen T. The general Copied and Local rules determine whether a type
is a read result.

## Example: read one version

```go
type Settings struct {
    im.Frozen
    Title im.FrozenString
    Limit int
}

type Cache struct {
    supervise.Service[supervise.NoArgs]
    current SettingsCell
}

func (c *Cache) Settings() SettingsProjection { return c.current.Load() }
```

Generate SettingsCell and SettingsProjection with the [projection generator](projections.md).
A caller reads fields through typed methods. It can keep copied
numbers or explicit string clones. It cannot keep the returned projection in
persistent storage.

For a result that combines computed values with projections, use [im.Local](local.md).
The [model service](../examples/models/fetcher.go) returns a name projection and a score
from one selected model. The [configuration service](../examples/configuration/limiter.go)
returns copied quota counters.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `string` | `im.StringProjection` from `im.Text(im.ProjectionOf(text))`. |
| Frozen record with string or collection fields | A local projection of that record. |
| `im.FrozenString`, `im.FrozenSlice[E]`, or `im.FrozenMap[K,V]` | The corresponding StringProjection, SliceProjection, or MapProjection. |
| Pointer, native container, interface, or Cell | A read-result value with the required data. |
| `(int, bool)` | A Frozen record with those two Copied fields. |

Ordinary function results still use their general type rules. Service inputs,
locks, and implementation code keep ordinary Go rules except for protected im
operations.
