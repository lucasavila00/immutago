# Service runtime

A service embeds `supervise.Service[A]`. The runtime calls Init and Run.
Other method calls run in the caller's goroutine and can overlap with Run and
other calls. Use the [service guide](../guides/services.md) for a walkthrough.

## Service contract

| Item | Requirement and example |
| --- | --- |
| Service type | A concrete struct embeds `supervise.Service[A]` through value fields; see the embedding forms below. |
| Setup | `*Worker` supplies `Init() error`, or uses the default Init from Service[A]. |
| Background work | `*Worker` supplies `Run(context.Context)`. |
| Start arguments | `s.Args()` reads A. The runtime saves the same arguments for each restart. Ordinary Go types are allowed; protected arguments require [Frozen types](types.md). |
| No arguments | Embed `supervise.Service[supervise.NoArgs]` and pass `supervise.NoArgs{}` to Start. |
| Exported methods | Use the [service result signatures](service-results.md), including promoted methods. Init, Run, and runtime methods are exempt. |
| Fields, inputs, closures, goroutines, and construction | Ordinary Go rules; uses of protected data keep the [im source rules](common-forms.md). |
| Reference calls | Use the complete [service-reference forms](service-references.md). |

Start requires the Init and Run signatures through Go's type
checker. The analyzer checks exported results, reference targets, direct Borrow
calls, and protected im operations. Their linked references give the complete
allowed forms; unlisted protected operations and direct Borrow forms are rejected.
Other service code follows ordinary Go rules. Runtime obligations below, including
returning when a context ends, require implementation review.

| Rejected examples — non-exhaustive | Allowed replacement |
| --- | --- |
| `type Worker struct { *supervise.Service[Args] }` | Embed `supervise.Service[Args]` as a value. |
| `type Pointer = *supervise.Service[Args]`; `type Worker struct { Pointer }` | Alias `supervise.Service[Args]` without `*`. |
| `type Worker struct { *Base }` where Base embeds Service | Embed `Base` as a value. |
| `supervise.Ref[int]` | Use a concrete service struct as the type argument. |
| `func (s *Worker) Name() string` | Return `im.StringProjection`; see [service results](service-results.md). |
| Save a Local projection in start arguments | Supply Frozen data and create a projection when reading. |

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

## Startup

| Operation or event | Behavior |
| --- | --- |
| `ref, err := supervise.Start[Worker](parent, args)` | Create a fresh Worker, call Init, then make it available through Ref and schedule Run. Only Worker needs an explicit type argument. |
| First Init returns an error | Start returns a zero Ref and the error. |
| First Init panics | The panic reaches the caller. |
| Start under a stopped group | Init still runs. If it succeeds, Run receives an already-cancelled context. Start waits for cleanup, then returns the initialized Ref. |
| Stop while initial Init runs | Stop waits only for registered services. If Init succeeds after Stop, Start waits for its cancelled Run and child cleanup. |
| Nil or zero Group passed to Start or StartGroup | Panic with `supervise: nil or zero Group` before Init. |
| Pointer on the selected Service embedding path | Panic before Init with `supervise: embed Service by value through each embedded field`. |

Init must not wait for an unstarted peer or child Run. For readiness that depends
on Run, expose a method such as `Ready() bool`. Init has no cancellation context.
It must return and must release application resources that it opened if setup fails.
Every successful Init gets exactly one Run call. Run can use defer to release those
resources. Check its context before starting work: it can already be cancelled.

## Restarts and failure

| Event | Behavior |
| --- | --- |
| Run returns normally | The service ends and its group removes it. |
| Run panics | Wait, create a new struct with the same arguments, call Init, and start Run after setup succeeds. |
| Restart Init fails | Count a failed attempt. The reference still selects the previous successfully initialized value. |
| A direct service method panics | The panic stays in its caller's goroutine. A call from Run or restart Init can fail that attempt. |
| One service fails six times within five minutes | Report the selected terminal `*supervise.Failure`, stop its entire root tree, then panic with that Failure. |
| Run or restart Init calls `runtime.Goexit()` | Report a terminal failure and stop the root tree. Do not retry. |
| A child fails under nested groups or services | Only that service retries. Exhaustion goes directly to the root; parent services and groups add no retries. |

The delay is one second after the first failure. It doubles for each restart in
the last five minutes. Borrow continues to select the last published value during
a delay and after stop. A call in progress keeps its selected value. Each service
has its own retry count. Groups do not restart.

## Child groups

| Parent | Lifetime |
| --- | --- |
| `root := supervise.NewRoot(report)` | Application group, until root.Stop. Pass nil to disable attempt reports; selected terminal failures still go to standard error. |
| `group := supervise.StartGroup(parent)` | Lifetime group; stop these services and their children together. |
| `s.Children()` | Children of one service run; a restart creates a new group. |

Create groups with these APIs. A nil or zero Group cannot run or stop. Groups otherwise follow
ordinary Go rules. A stopped parent supplies a stopped nested group.

### Child lifetimes

| Event | Effect |
| --- | --- |
| Parent Init returns an error or panics | Stop children from that setup attempt. Application resources remain Init's responsibility. |
| A method starts a child before parent Run begins | Child Init runs immediately. Child Run waits for normal activation or receives a cancelled context during cleanup. |
| Parent is stopped | Stop children last started first and wait for them before cancelling the parent's Run context. |
| Parent Run returns or panics | Stop and wait for children before the run ends or the restart delay begins. |
| A child exceeds its restart limit | Stop the whole root tree. No parent retries that failure. |
| A method starts a child after the parent run ends | Init can succeed. The call waits for the child's cancelled Run, then returns its reference. A saved group from that run uses the same rule. |

Create children again in each parent run. An old child Ref
still selects the old child after a parent restart. Keep child references in
parent fields and expose methods that select the current child. The checker
permits Ref copies; it does not prove their lifetime.

Services and groups can be started after application startup. Put a dynamic
service under `StartGroup(parent)` to stop that subtree separately later.
For a model swap, publish the new model reference before stopping the old group.
Stop does not wait for requests already using the old reference.

## Shutdown

| Operation | Behavior |
| --- | --- |
| `done := root.Stop()` | Request shutdown without waiting. Repeated calls return the same channel. |
| Stop on a nil or zero Group | Panic with `supervise: nil or zero Group`. |
| `<-root.Stop()` | Wait until all children stop. |
| `<-group.Stop()` on a nested group | Also wait for removal from the parent. Other groups continue. |
| Stop is requested | Prevent new retries in that subtree immediately; cancel running services in reverse start order. |
| Restart Init is already running when Stop is requested | Let Init finish. If it succeeds, stop its children and call Run with an already-cancelled context. Do not publish the replacement. |

Shutdown gives pending initialized services a cancelled Run through the same
watcher used for active services. It waits for children in reverse start order
before it enters a parent's cancelled Run. It does not activate staged children
with live contexts. Cleanup panics are reported; cleanup Goexit ends its watcher.

Stop does not interrupt work already admitted. Registered work must finish
before the completion channel closes. Run must return when its context ends.
Stop cannot force Run or Init to return
and does not wait for direct method calls. Release locks that children need
before waiting for Stop. A child or report function must not wait for its own
group to stop: the group is waiting for that code to finish. A child can call a
parent during shutdown, but must not wait for a parent method that waits for
that child's group to stop.

## Failure reports

The report argument to NewRoot has type `func(value any, stack []byte)`:

- It receives each recovered panic or restart Init error before the delay, and each Goexit failure.
- It also receives the selected terminal `*Failure` before the root waits for shutdown.
- Nested groups use the same function. Each failed attempt is reported once; the
  selected terminal failure is a separate report.
- Reports for different children can run concurrently.
- It must return. The group waits for it and does not recover its panics.

The root's terminal panic is `*supervise.Failure`:

| Field | Content |
| --- | --- |
| `Service` | Go type string of the failing service. |
| `Value` | Original panic, restart Init error, or Goexit error. |
| `Stack` | Stack from the failing service's goroutine. |

In the report callback, use `errors.As` on an error value to identify Failure and
`errors.Is` to match its original error. A nil callback suppresses attempt reports,
but the runtime writes the selected terminal Failure and its original stack to
standard error before cleanup. A blocked service cannot hide that report.

The root selects one terminal failure. Ordered shutdown completes before it
panics. A Stop request for the failing service's subtree, accepted first, prevents that
failure from terminating the process. Stopping an unrelated group does not.
The final panic runs in the root's shutdown goroutine; a caller cannot recover it.
