# Service runtime contract

A service embeds `supervise.Service[A]`. The runtime calls Init and Run.
Other method calls run in the caller's goroutine and can overlap with Run and
other calls. See the [service guide](../../docs/guides/services.md) for an example
and the [API](readme.md) for signatures.

| Item | Requirement |
| --- | --- |
| Service declaration | Use the [value embedding forms](../../docs/reference/service-results.md#service-embedding--exhaustive). |
| Setup | `*Worker` supplies `Init(*Supervisor) error`, or uses the default Init from Service[A]. |
| Background work | `*Worker` supplies `Run(context.Context)`. |
| Start arguments | `s.Args()` reads A. Start saves these arguments on the service. Use `supervise.NoArgs{}` when no arguments are needed. |
| Exported methods | Use the [result rules](../../docs/reference/service-results.md). |
| Mutable state and inputs | Review field access, locks, saved inputs, and dependency calls in the service implementation. |

## Copy contract

| Value | Copy rule |
| --- | --- |
| `Service[A]` | Contains only the saved arguments. Copies follow the Go copy and synchronization rules for A. |
| Service struct | Copy only when its arguments and application fields permit it. For example, keep an initialized mutex or Cell behind a pointer. |
| Saved `*Supervisor` from Init | Copies share the same supervisor and owner lifetime. Save it or pass it to other code. |
| Supervisor | Do not copy an initialized Supervisor or a struct or array that contains it, including after Stop. Pass its pointer. |
| Ref | Copies select the same original service. |

Start calls Init and Run on the service it creates. A service copy does not start
another Run or extend the original service's lifetime. Value receivers follow the
same field rules as explicit copies. Standard `go vet ./...` reports common
mutex and Supervisor copies; im rules still apply to protected fields.

## References

Ref is a Frozen struct with one UncheckedValue field. Copies select the same
service; its fields stay fixed while the service itself can change.

| Operation | Result |
| --- | --- |
| `ref.Borrow()` | The `*T` created by Start, including after Stop. Save it, return it, or call its methods using ordinary Go rules. No allocation. |
| `ref.IsZero()` | Whether the reference is unset. Borrow panics on a zero Ref. No allocation. |
| Copy a Ref | Another value that selects the same service. |
| Stop the service | End its supervised work. The pointer stays valid; Stop does not prevent or wait for method calls. |

## Startup

| Operation or event | Behavior |
| --- | --- |
| `ref, err := supervise.Start[Worker](parent, args)` | Check admission, create a fresh Worker, call Init with its own supervisor, then register it and return Ref. Run is asynchronous. |
| Init returns an error | Start returns a zero Ref and the error. |
| Init panics | The panic reaches the caller. |
| Start after the supervisor or an ancestor closes admission | Return a zero Ref and `context.Canceled`. Do not call Init or Run. |
| Stop after Start is accepted | Wait for Init before stopping registered services. Successful Init still registers and returns Ref; Run can receive a cancelled context. |
| Nil or zero Supervisor passed to Start or StartSupervisor | Panic with `supervise: nil or zero Supervisor` before Init. |
| Pointer on the selected Service embedding path | An accepted Start panics before Init with `supervise: embed Service by value through each embedded field`. |

Go infers A from the argument's type. An untyped constant supplies its default
type. If that type differs from the service's A, specify A or convert the argument:

| Service declaration | Specify A | Convert the argument |
| --- | --- | --- |
| `type Indexer struct { supervise.Service[fs.FS] }` | `supervise.Start[Indexer, fs.FS](root, fstest.MapFS{})` | `supervise.Start[Indexer](root, fs.FS(fstest.MapFS{}))` |
| `type Counter struct { supervise.Service[int64] }` | `supervise.Start[Counter, int64](root, 5)` | `supervise.Start[Counter](root, int64(5))` |

After setup, the runtime admits Run and activates its staged child supervisors in one
decision ordered against Stop. An admitted Run can enter its function after Stop
is requested; its context still ends at its turn in shutdown.

Init must not wait for an unstarted peer or child Run. For readiness that depends
on Run, expose a method such as `Ready() bool`. Init has no cancellation context.
It must return without waiting for its owning supervisor or an ancestor to stop, or
for cancellation that needs that shutdown. Init can stop and wait for the
supervisor passed to it to undo setup. It must release application resources that it opened
if setup fails.

Init must return if a dependency stops answering. Bound each blocking operation
with an application timeout or a context cancelled independently of tree shutdown.
A called API must also return when its context ends.

Every successful Init gets exactly one Run call. Run can use defer to release those
resources. Check its context before starting work: it can already be cancelled.
Successful Start means that Init completed; it does not mean that Run is still active.

## Completion and failure

| Event | Behavior |
| --- | --- |
| Run returns normally | Stop its children, then remove the service from its supervisor. Other services continue. |
| Run receives cancellation | Return normally to end without a failure. |
| Run panics | Report the original panic and stack. Stop the entire root tree, wait for accepted Init calls and cleanup, then panic with `*supervise.Failure`. |
| Run calls `runtime.Goexit()` | Stop the root through the same failure path. Failure.Value is an error with the text `supervise: service called runtime.Goexit`. |
| A child fails under nested supervisors or services | Its failure goes directly to the root. |
| A direct service method panics | The panic stays in its caller's goroutine. If called from Run, it fails that Run. |
| Run panics or calls Goexit during shutdown or cleanup | Use the same terminal failure path. Stop and a parent's normal return cannot suppress a failure. |
| Several Run failures occur | Keep the first failure recorded by the runtime. Report each failure once. |
| Terminal panic | It runs in a runtime goroutine after accepted Init calls, reporting, and cleanup. A caller cannot recover it. |

Each accepted Start creates one service value and calls Run once after successful Init.
Borrow returns that same value, including after it stops.

## Supervisors

| Parent | Lifetime |
| --- | --- |
| `root := supervise.NewRoot(report)` | Application supervisor, until root.Stop. Pass nil to write failure reports and their original stacks to standard error. |
| `supervisor := supervise.StartSupervisor(parent)` | Lifetime supervisor; stop these services and their children together. |
| `Init(supervisor *supervise.Supervisor)` | Runtime supplies a supervisor owned by this service. Save the pointer to start services later. It stops when this service ends. |

Use the supervisor passed to Init or create one with the functions above. Keep its pointer. A nil or zero Supervisor cannot
run or stop. Supervisors follow the copy contract above and ordinary Go rules.
A stopped parent supplies a stopped nested supervisor.

### Child lifetimes

| Event | Effect |
| --- | --- |
| Parent Init returns an error, panics, or calls Goexit | Stop children from that setup. Application resources remain Init's responsibility. |
| An accepted child Start before parent Run is admitted | Child Init runs immediately. Child Run waits for normal activation or receives a cancelled context during cleanup. |
| Stop races with staged child activation | Activation completes before Stop or leaves the children pending. In both cases, children wait for their reverse-order shutdown turns. |
| Parent is stopped | Stop children in reverse registration order and wait for them before cancelling the parent's Run context. |
| Parent Run returns or panics | Stop and wait for children before the service ends. |
| A child Run panics or calls Goexit | Stop the whole root tree. |

Services and supervisors can be started after application startup. Put a dynamic
service under `StartSupervisor(parent)` to stop that subtree separately later.
For a model swap, publish the new model reference before stopping the old supervisor.
Stop does not wait for requests already using the old reference.

## Shutdown

| Operation | Behavior |
| --- | --- |
| `done := root.Stop()` | Request shutdown without waiting. Repeated calls return the same channel. |
| Stop on a nil or zero Supervisor | Panic with `supervise: nil or zero Supervisor`. |
| `<-root.Stop()` | Wait for accepted Init calls and all children to stop. A terminal failure prevents normal completion. |
| `<-supervisor.Stop()` on a nested supervisor | Wait for this supervisor's children and removal from its parent. This means the subtree has ended; a child failure still stops the root. |
| Stop is requested | Close admission immediately. Each supervisor waits for its accepted Init calls, then cancels children in reverse registration order. |
| A registered service has not been admitted to Run when Stop is requested | Wait for its turn, stop its children, then call Run with an already-cancelled context. |

Cleanup panics and Goexit are terminal failures.
Services register after Init; concurrent Start calls can register in a different
order from their invocation order. Nested supervisors register when StartSupervisor is called.

Stop does not interrupt work already admitted. Registered work must finish
before the completion channel closes. Run must return when its context ends.
Stop cannot force Run or Init to return
and does not wait for direct method calls. Release locks that children need
before waiting for Stop. A service must not wait from Run for the supervisor
that contains it: that supervisor is waiting for Run to finish. The supervisor
passed to Init contains only the services it starts; it can stop and wait for
that supervisor. A report function must not wait for tree shutdown. A child can call a
parent during shutdown, but must not wait for a parent method that waits for
that child's supervisor to stop.

## Failure reports

The report argument to NewRoot has type `func(value any, stack []byte)`:

| Event or item | Behavior |
| --- | --- |
| Report payload | Each recovered Run panic or Goexit supplies one `*Failure` and its original stack. |
| Failure selection | Record the first terminal failure before calling the reporter. |
| An accepted Init or child cleanup blocks | Reporting does not wait for it. The terminal panic does. |
| Several failures | Reports can overlap. Nested supervisors use the same callback. |
| Callback calls Stop | It can request Stop, but must not wait for completion. It cannot suppress the failure. |
| Callback returns | Required before its service completes. Callback panics are not recovered. |
| Nil callback | Write each Failure and its original stack to standard error. |

The root's terminal panic is `*supervise.Failure`:

| Field | Content |
| --- | --- |
| `Service` | Go type string of the failing service. |
| `Value` | Original panic or Goexit error. |
| `Stack` | Stack from the failing service's goroutine. |

`supervisetest.NewRoot(t)` adds automatic test cleanup with a 5-second timeout.
It uses the same terminal failure rule. Use a subprocess to test Run panics or Goexit.
