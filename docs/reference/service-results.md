# Service results

Each result of an exported service method must be a read result or Go's
predeclared `error` type. The same rules apply to `Service[A]` and
`TerminalService[A]`. There is no limit on the number or order of results.
Init, Run, and runtime methods are exempt. The rule also covers promoted methods.

| Result signatures — exhaustive | Example |
| --- | --- |
| No result | `func (p *Pool) Refresh()` |
| One or more read results or errors, in any order | `func (p *Pool) Count() int`, `func (p *Pool) Lookup() (Reply, bool)`, `func (p *Pool) Read() (error, Reply, int, error)` |
| Named results with the same type rule | `func (p *Pool) Lookup() (value Reply, found bool, err error)` |

Each result is checked separately. A tuple has no record type and needs no
marker. A record used as one result still needs its normal marker and field proofs.
Unlisted result types are rejected.

An error result uses Go's predeclared `error` type; aliases work. Error values
follow ordinary Go rules. Callers can save errors, and custom errors can hold
ordinary mutable state. The checker does not inspect error origins or payloads.
General im checks still reject boxing protected storage in an error.

`error` is not a read result or an allowed field in a marked record. A
[read result](types.md) uses this complete list:

| Read-result types — exhaustive | Example |
| --- | --- |
| Copied scalar or unnamed array | `bool`, `int`, `[2]int` |
| Copied record or unnamed array of Copied records | `Score` with `im.Copied`, `[2]Score` |
| Unchecked handle or unnamed array of handles | `im.UncheckedValue[Model]`, `[2]im.UncheckedValue[Model]` |
| Service reference | `supervise.Ref[Worker]` |
| Local projection | Generated `SettingsProjection`, `im.StringProjection`, `im.MapProjection[string, int]` |
| Local record or unnamed array of read results | `Usage` with `im.Local` and numeric fields, `[2]im.StringProjection` |

There is no separate field rule for services. Cell[T] and Projection[T] require the
same Frozen T. The shared read-result rule checks Copied values, Local values, unchecked handles,
and unnamed arrays recursively. Frozen records are not read results, even when all fields are
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

Each result keeps its normal type, write, and storage rules. Direct typed helpers
use the same [binding forms](common-forms.md#statements-and-calls).
For example, a Local result needs a local name or blank, while an ordinary bool
result can be assigned to an ordinary field:

```go
var reply Reply
reply, state.Found = pool.Borrow().Lookup()
```

Named results, explicit returns, and forwarded returns use the same checks:

```go
func (p *Pool) Named() (reply Reply, found bool) {
    reply, found = p.Lookup()
    return
}
func (p *Pool) Forward() (Reply, bool) { return p.Lookup() }
```

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `string` | `im.StringProjection` from `im.ProjectString(text)`. |
| `Usage` with `im.Frozen`, even when it has only numeric fields | Use im.Local for the result, or return its generated projection. |
| `im.Frozen`, `im.Local`, or `im.Copied` marker | Return no result when no data is needed; embed im.Local in a result record when data is needed. |
| `im.FrozenString`, `im.FrozenSlice[E]`, or `im.FrozenMap[K,V]` | The corresponding StringProjection, SliceProjection, or MapProjection. |
| Pointer, native container, interface other than `error`, or Cell in any result position | A read-result value with the required data. |
| `(Reply, string, bool)` | Return `(Reply, im.StringProjection, bool)`. |
| A distinct `type Failure error` in any result position | Declare the result as `error` or an alias such as `type Failure = error`. |
| `savedReply, found = pool.Borrow().Lookup()` with a package-level Local `savedReply` | Bind the Local result to a local name; save only ordinary results. |

Ordinary function results still use their general type rules. Service inputs,
locks, and implementation code keep ordinary Go rules except for protected im
operations.
