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
| Runtime ownership | Do not copy an initialized service or Group. Declare service methods with pointer receivers and pass service and group pointers; see the copy contract below. |
| Start arguments | `s.Args()` reads A. Start saves these arguments on the service. Ordinary Go types are allowed; protected arguments require [Frozen types](types.md). |
| No arguments | Embed `supervise.Service[supervise.NoArgs]` and pass `supervise.NoArgs{}` to Start. |
| Exported methods | Use the [service result signatures](service-results.md), including promoted methods. Init, Run, and runtime methods are exempt. |
| Fields, inputs, closures, goroutines, and construction | Ordinary Go rules; uses of protected data keep the [im source rules](common-forms.md). |
| Reference calls | Use the complete [service-reference forms](service-references.md). |

Start requires the Init and Run signatures through Go's type
checker. The analyzer checks exported results, handle fields, and protected im
operations. Their linked references give the complete allowed forms; unlisted
protected operations are rejected.
Other service code follows ordinary Go rules. Runtime obligations below, including
returning when a context ends, require implementation review.

### Copy contract

Keep initialized services and Groups behind pointers, including after Stop.
Use pointer receivers. Do not copy their values or containing structs and arrays.
Ref values can be copied.

UnsafeChildren requires the original service pointer created by Start. The caller
must obey this Go contract. Standard `go vet ./...` reports common owner copies;
immutago does not enforce this contract.

| Rejected examples — non-exhaustive | Allowed replacement |
| --- | --- |
| `type Worker struct { *supervise.Service[Args] }` | Embed `supervise.Service[Args]` as a value. |
| `type Pointer = *supervise.Service[Args]`; `type Worker struct { Pointer }` | Alias `supervise.Service[Args]` without `*`. |
| `type Worker struct { *Base }` where Base embeds Service | Embed `Base` as a value. |
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
| Init returns an error | Start returns a zero Ref and the error. |
| Init panics | The panic reaches the caller. |
| Start under a stopped group | Init still runs. If it succeeds, Run receives an already-cancelled context. Start waits for cleanup, then returns the initialized Ref. |
| Stop while Init runs | Stop waits only for registered services. If Init succeeds after Stop, Start waits for its cancelled Run and child cleanup. |
| Nil or zero Group passed to Start or StartGroup | Panic with `supervise: nil or zero Group` before Init. |
| Pointer on the selected Service embedding path | Panic before Init with `supervise: embed Service by value through each embedded field`. |

After setup, the runtime admits Run and activates its staged child groups in one
decision ordered against Stop. An admitted Run can enter its function after Stop
is requested; its context still ends at its turn in shutdown.

Init must not wait for an unstarted peer or child Run. For readiness that depends
on Run, expose a method such as `Ready() bool`. Init has no cancellation context.
It must return and must release application resources that it opened if setup fails.
Every successful Init gets exactly one Run call. Run can use defer to release those
resources. Check its context before starting work: it can already be cancelled.

## Completion and failure

| Event | Behavior |
| --- | --- |
| Run returns normally | Stop its children, then remove the service from its group. Other services continue. |
| Run panics | Report the original panic and stack. Stop the entire root tree, wait for cleanup, then panic with `*supervise.Failure`. |
| Run calls `runtime.Goexit()` | Stop the root through the same failure path; Failure.Value identifies Goexit. |
| A child fails under nested groups or services | Its failure goes directly to the root. |
| A direct service method panics | The panic stays in its caller's goroutine. If called from Run, it fails that Run. |
| Stop of the failing subtree is accepted before the failure | Report the failure and complete shutdown without a terminal root panic. |

Each Start creates one service value and calls Run once after successful Init.
Borrow returns that same value, including after it stops.

## Child groups

| Parent | Lifetime |
| --- | --- |
| `root := supervise.NewRoot(report)` | Application group, until root.Stop. Pass nil to disable individual Run reports; selected terminal failures still go to standard error. |
| `group := supervise.StartGroup(parent)` | Lifetime group; stop these services and their children together. |
| `s.UnsafeChildren()` | Child group owned by this service. Use the original pointer created by Start. |
| `new(Worker).UnsafeChildren()` | Panic with `supervise: nil or zero Group` in the caller. Use Start to create a service that needs child groups. |

Create groups with these APIs and keep their pointers. A nil or zero Group cannot
run or stop. Groups follow the copy contract above and ordinary Go rules.
A stopped parent supplies a stopped nested group.

### Child lifetimes

| Event | Effect |
| --- | --- |
| Parent Init returns an error or panics | Stop children from that setup. Application resources remain Init's responsibility. |
| A method starts a child before parent Run is admitted | Child Init runs immediately. Child Run waits for normal activation or receives a cancelled context during cleanup. |
| Stop races with staged child activation | Activation completes before Stop or leaves the children pending. In both cases, children wait for their reverse-order shutdown turns. |
| Parent is stopped | Stop children last started first and wait for them before cancelling the parent's Run context. |
| Parent Run returns or panics | Stop and wait for children before the service ends. |
| A child Run panics or calls Goexit | Stop the whole root tree. |
| A method starts a child after the parent run ends | Init can succeed. The call waits for the child's cancelled Run, then returns its reference. A saved group from that run uses the same rule. |

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
| Stop is requested | Close admission immediately; cancel services in reverse start order. |
| A registered service has not been admitted to Run when Stop is requested | Wait for its turn, stop its children, then call Run with an already-cancelled context. |

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

- It receives each recovered Run panic or Goexit before the service waits for child cleanup.
- It also receives the selected terminal `*Failure` before the root waits for shutdown.
- Nested groups use the same function. Each failed Run is reported once; the
  selected terminal failure is a separate report.
- Reports for different children can run concurrently.
- It must return. The group waits for it and does not recover its panics.

The root's terminal panic is `*supervise.Failure`:

| Field | Content |
| --- | --- |
| `Service` | Go type string of the failing service. |
| `Value` | Original panic or Goexit error. |
| `Stack` | Stack from the failing service's goroutine. |

In the report callback, use `errors.As` on an error value to identify Failure and
`errors.Is` to match its original error. A nil callback suppresses individual Run reports,
but the runtime writes the selected terminal Failure and its original stack to
standard error before cleanup. A blocked service cannot hide that report.

The root selects one terminal failure. Ordered shutdown completes before it
panics. A Stop request for the failing service's subtree, accepted first, prevents that
failure from terminating the process. Stopping an unrelated group does not.
The final panic runs in the root's shutdown goroutine; a caller cannot recover it.
