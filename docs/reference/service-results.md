# Service declarations and results

A service is a struct that embeds `supervise.Service[A]`, directly or through
another struct. Its fields, value copies, inputs, and methods other than Init and
Run use ordinary Go rules. Protected im data keep their normal type and source rules.

## Service embedding — exhaustive

| Form | Example |
| --- | --- |
| Direct value embedding | `type Worker struct { supervise.Service[Args] }` |
| Alias to a value type | `type Base = supervise.Service[Args]`; `type Worker struct { Base }` |
| Alias to a service struct | `type Alias = Worker` |
| Defined struct with the same value fields | `type Another Worker`; declare its own Run method. |
| Anonymous struct with value embedding | `type Anonymous = struct { Worker }` |
| Value embedding through another struct | `type Base struct { supervise.Service[Args] }`; `type Worker struct { Base }` |

The Service embedding used by Start must use value fields.
Unlisted embedding paths are rejected. Other fields follow ordinary Go rules.

Start saves its arguments. Ordinary Go types are allowed. Protected arguments
must be Frozen; Local values are rejected.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `type Worker struct { *supervise.Service[Args] }` | Embed `supervise.Service[Args]` by value. |
| `type Pointer = *supervise.Service[Args]`; `type Worker struct { Pointer }` | Alias `supervise.Service[Args]` without `*`. |
| `type Worker struct { *Base }` where Base embeds Service | Embed `Base` by value. |
| Local projection in Start arguments | Supply Frozen data and create a projection when reading. |

## Lifecycle receivers — exhaustive

The selected Init and Run methods must have declared pointer receivers. This rule
applies to every named or anonymous service type, before any Start call. Go checks
the required signatures and method ambiguity at Start. A base service type can
omit Run until another type supplies it.

| Allowed receiver form | Example |
| --- | --- |
| Declared pointer method | `func (w *Worker) Init(*supervise.Supervisor) error { return nil }`; `func (w *Worker) Run(context.Context) {}` |
| Alias of a pointer receiver | `type Receiver = *Worker`; `func (Receiver) Run(context.Context) {}` |
| Default Init | `type Worker struct { supervise.Service[Args] }` uses `(*Service[Args]).Init`. |
| Pointer method promoted through value fields | `type Base struct { supervise.Service[Args] }`; `func (*Base) Run(context.Context) {}`; `type Worker struct { Base }` |
| Pointer method promoted through pointer fields outside the Service embedding path | `type Hooks struct{}`; `func (*Hooks) Run(context.Context) {}`; `type Worker struct { supervise.Service[Args]; *Hooks }` |
| Aliased, defined, or generic service | `type Alias = Worker`; `type Other Worker` with its own pointer Run; `type Worker[A any] struct { supervise.Service[A] }` with `func (*Worker[A]) Run(context.Context) {}` |
| Anonymous service with promoted pointer methods | `type Anonymous = struct { Base }`, where Base supplies pointer Init and Run methods. |

Unlisted receiver forms are rejected.
Promotion uses the receiver in the original method declaration. An interface
method has no declared pointer receiver. A direct pointer method can replace a
promoted value or interface method and forward its call when needed.

The Service embedding path still requires value fields. Other embedded pointers
use ordinary Go initialization and synchronization contracts.
Ordinary value methods and explicit service copies keep
the [copy contract](../../pkg/supervise/contract.md#copy-contract).

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `func (w Worker) Run(ctx context.Context) {}` | `func (w *Worker) Run(ctx context.Context) {}`; lock shared fields before reading them. |
| `func (w Worker) Init(*supervise.Supervisor) error { w.limit = 5; return nil }` | Use `*Worker` so setup writes change the original service. |
| A value Run promoted from `Hooks` or `*Hooks` | Declare `func (*Hooks) Run(context.Context)`, or declare a pointer Run on Worker. |
| Run promoted from `type Runner interface { Run(context.Context) }` | Declare `func (w *Worker) Run(ctx context.Context) { w.Runner.Run(ctx) }`. |

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
