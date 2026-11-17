# Service declarations and results

A service is a struct that embeds `supervise.Service[A]`, directly or through
another struct. Its fields, value copies, method receivers, and inputs use ordinary
Go rules. Protected im data keep their normal type and source rules.

## Service embedding — exhaustive

| Form | Example |
| --- | --- |
| Direct value embedding | `type Worker struct { supervise.Service[Args] }` |
| Alias to a value type | `type Base = supervise.Service[Args]`; `type Worker struct { Base }` |
| Alias to a service struct | `type Alias = Worker` |
| Defined struct with the same value fields | `type Another Worker`; declare its own Run method. |
| Anonymous struct with value embedding | `type Anonymous = struct { Worker }` |
| Value embedding through another struct | `type Base struct { supervise.Service[Args] }`; `type Worker struct { Base }` |

Every field on the selected path to the runtime methods must be a value field.
Unlisted embedding paths are rejected. Other fields follow ordinary Go rules.

Start saves its arguments. Ordinary Go types are allowed. Protected arguments
must be Frozen; Local values are rejected.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `type Worker struct { *supervise.Service[Args] }` | Embed `supervise.Service[Args]` by value. |
| `type Pointer = *supervise.Service[Args]`; `type Worker struct { Pointer }` | Alias `supervise.Service[Args]` without `*`. |
| `type Worker struct { *Base }` where Base embeds Service | Embed `Base` by value. |
| Local projection in Start arguments | Supply Frozen data and create a projection when reading. |

## Exported method results

Each result of an exported service method must be a read result or Go's
predeclared `error` type. There is no limit on the number or order of results.
Init, Run, and runtime methods are exempt. The rule also covers promoted methods.

| Result signatures — exhaustive | Example |
| --- | --- |
| No result | `func (p *Pool) Refresh()` |
| One or more read results or errors, in any order | `func (p *Pool) Count() int`, `func (p *Pool) Lookup() (Reply, bool)`, `func (p *Pool) Read() (error, Reply, int, error)` |
| Named results with the same type rule | `func (p *Pool) Lookup() (value Reply, found bool, err error)` |

Each result is checked separately. A tuple has no record type and needs no
marker. A record used as one result still needs its normal marker and field types.
Unlisted result types are rejected.

An error result uses Go's predeclared `error` type; aliases work. Error values
follow ordinary Go rules. Callers can save errors, and custom errors can hold
ordinary mutable state. The checker does not inspect error origins or payloads.
Local values cannot be boxed in an error.

The complete [read-result list](types.md#read-results--exhaustive) also applies
to Local fields. `error` is not a read result or a Frozen, Local, or Copied field type.

## Bind results

Each result keeps its normal type, write, and storage rules. Direct typed helpers
use the same [binding forms](common-forms.md#statements-and-calls).
For example, a Local result needs a local name or blank, while an ordinary bool
result can be assigned to an ordinary field:

```go
var reply Reply
reply, state.Found = pool.Lookup()
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
| `*Usage` with `im.Local`, `[]int`, `any`, or `im.Cell[int]` in any result position | A read-result value with the required data. |
| `(Reply, string, bool)` | Return `(Reply, im.StringProjection, bool)`. |
| A distinct `type Failure error` in any result position | Declare the result as `error` or an alias such as `type Failure = error`. |
| `savedReply, found = pool.Lookup()` with a package-level Local `savedReply` | Bind the Local result to a local name; save only ordinary results. |

Ordinary function results still use their general type rules. Service inputs,
locks, and implementation code keep ordinary Go rules except for protected im
operations.
