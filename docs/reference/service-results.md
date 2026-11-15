# Service results

Each exported service method uses one of these signatures. Init, Run, and
runtime methods are exempt. The rule also covers promoted methods.

| Result signatures — exhaustive | Example |
| --- | --- |
| No result | `func (p *Pool) Refresh()` |
| One read result | `func (p *Pool) Count() int` |
| `error` | `func (p *Pool) Restart() error` |
| One read result, then `error` | `func (p *Pool) Read() (Reply, error)` |

The error slot uses Go's predeclared `error` type; aliases work. Error values
follow ordinary Go rules. Callers can save errors, and custom errors can hold
ordinary mutable state. The checker does not inspect error origins or payloads.
General im checks still reject boxing protected storage in an error.

`error` is not a read result or an allowed field in a marked record. A
[read result](types.md) is Copied or Local:

| Read-result types — exhaustive | Example |
| --- | --- |
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

## Bind results

```go
reply, err := pool.Borrow().Read()
if err != nil { return err }
use(reply)
```

Protected result pairs use two local names or blanks with `:=`, `=`, or local
`var`. Direct typed helpers use the same [binding forms](common-forms.md#statements-and-calls).
Ordinary results such as `(int, error)` keep ordinary Go binding rules.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `string` | `im.StringProjection` from `im.ProjectString(text)`. |
| `Usage` with `im.Frozen`, even when it has only numeric fields | Use im.Local for the result, or return its generated projection. |
| `im.Frozen`, `im.Local`, or `im.Copied` marker | Return no result when no data is needed; embed im.Local in a result record when data is needed. |
| `im.FrozenString`, `im.FrozenSlice[E]`, or `im.FrozenMap[K,V]` | The corresponding StringProjection, SliceProjection, or MapProjection. |
| Pointer, native container, interface other than the error slot, or Cell | A read-result value with the required data. |
| `(Reply, error, error)` or `(error, Reply)` | Return `(Reply, error)`. |
| A distinct `type Failure error` as the error slot | Declare the result as `error` or an alias such as `type Failure = error`. |
| `(int, bool)` | `type Result struct { im.Copied; Count int; OK bool }`. |

Ordinary function results still use their general type rules. Service inputs,
locks, and implementation code keep ordinary Go rules except for protected im
operations.
