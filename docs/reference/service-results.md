# Service results

Each exported service method returns no result or one [read result](types.md):
Copied data or a Local value. Init, Run, and runtime methods are exempt.
The rule also covers promoted methods.

| Allowed result — exhaustive | Example |
| --- | --- |
| No result | `func (c *Cache) Refresh()` |
| Copied scalar or unnamed array | `bool`, `int`, `[2]int` |
| Copied record or unnamed array of Copied records | `Score` with `im.Copied`, `[2]Score` |
| Service reference | `supervise.Ref[Worker]` |
| Local projection | Generated `SettingsProjection`, `im.StringProjection`, `im.MapProjection[string, int]` |
| Local record or unnamed array of read results | `Usage` with `im.Local` and numeric fields, `[2]im.StringProjection` |

There is no separate field rule for services. Cell[T] and Projection[T] require the
same Frozen T. The general Copied and Local rules determine whether a type
is a read result. Frozen records are not read results, even when all fields are
numbers. Use [im.Copied](copied.md) for an independent value record, or im.Local
for a record that includes projections. Neither marker requires generation.

For example, a service with a generated SettingsCell returns its projection:

```go
func (c *Cache) Settings() SettingsProjection { return c.current.Load() }
```

For computed values combined with projections, use [im.Local](local.md).
The search indexer's [Status](../../examples/search/indexer.go) has a version
number, document count, and local error-text projection.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `string` | `im.StringProjection` from `im.ProjectString(text)`. |
| `Usage` with `im.Frozen`, even when it has only numeric fields | Use im.Local for the result, or return its generated projection. |
| `im.Frozen`, `im.Local`, or `im.Copied` marker | Return no result when no data is needed; embed im.Local in a result record when data is needed. |
| `im.FrozenString`, `im.FrozenSlice[E]`, or `im.FrozenMap[K,V]` | The corresponding StringProjection, SliceProjection, or MapProjection. |
| Pointer, native container, interface, or Cell | A read-result value with the required data. |
| `(int, bool)` | `type Result struct { im.Copied; Count int; OK bool }`. |

Ordinary function results still use their general type rules. Service inputs,
locks, and implementation code keep ordinary Go rules except for protected im
operations.
