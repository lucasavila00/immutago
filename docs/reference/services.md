# Service runtime

A service embeds `supervise.Service[A]`. The runtime calls Init and Run.
Other method calls run in the caller's goroutine and can overlap with Run and
other calls. Use the [service guide](../guides/services.md) for a walkthrough.

## Service contract

| Item | Requirement and example |
| --- | --- |
| Service type | A concrete struct embeds `supervise.Service[A]`: `type Worker struct { supervise.Service[Args] }`. |
| Setup | `*Worker` supplies `Init() error`, or uses the default Init from Service[A]. |
| Background work | `*Worker` supplies `Run(context.Context)`. |
| Start arguments | `s.Args()` reads A. The runtime saves the same arguments for each restart. Ordinary Go types are allowed; protected arguments require [Frozen types](types.md). |
| No arguments | Embed `supervise.Service[supervise.NoArgs]` and pass `supervise.NoArgs{}` to Start. |
| Exported methods | Use the [service result signatures](service-results.md), including promoted methods. Init, Run, and runtime methods are exempt. |
| Fields, inputs, closures, goroutines, and construction | Ordinary Go rules; uses of protected data keep the [im source rules](common-forms.md). |
| Reference calls | Use the complete [service-reference forms](service-references.md). |

Start and StartReserved require the Init and Run signatures through Go's type
checker. The analyzer checks exported results, reference targets, direct Borrow
calls, and protected im operations. Their linked references give the complete
allowed forms; unlisted protected operations and direct Borrow forms are rejected.
Other service code follows ordinary Go rules. Runtime obligations below, including
returning when a context ends, require implementation review.

| Rejected examples — non-exhaustive | Allowed replacement |
| --- | --- |
| `supervise.Ref[int]` | Use a concrete service struct as the type argument. |
| `func (s *Worker) Name() string` | Return `im.StringProjection`; see [service results](service-results.md). |
| Save a Local projection in start arguments | Supply Frozen data and create a projection when reading. |

## Startup

| Operation or event | Behavior |
| --- | --- |
| `ref, err := supervise.Start[Worker](parent, args)` | Create a fresh Worker, call Init, then make it available through Ref and schedule Run. Only Worker needs an explicit type argument. |
| First Init returns an error | Start returns a zero Ref and the error. |
| First Init panics | The panic reaches the caller. |
| `ref := supervise.Reserve[Worker]()` | Create a nonzero reference for later startup; Borrow panics until Init succeeds. |
| `err := supervise.StartReserved(parent, ref, args)` | Initialize a reserved reference. An Init error leaves it unstarted and permits a retry. |
| StartReserved with a zero Ref | Panic with `supervise: StartReserved needs a Ref from Reserve`. Use `Reserve[Worker]()` to create the reference. |
| Start an already started reservation | Panic. With concurrent starts, the losing attempt stops its staged children and panics. |
| Start under a stopped group | Init still runs and can succeed; Run does not run. The runtime stops staged children. |

Init must not wait for an unstarted peer or child Run. For readiness that depends
on Run, expose a method such as `Ready() bool`. Init has no cancellation context.
It must return and must release application resources that it opened if setup fails.

## Restarts and failure

| Event | Behavior |
| --- | --- |
| Run returns normally | The service ends and its group removes it. |
| Run panics | Wait, create a new struct with the same arguments, call Init, and start Run after setup succeeds. |
| Restart Init fails | Count a failed attempt. The reference still selects the previous successfully initialized value. |
| A direct service method panics | The panic stays in its caller's goroutine. A call from Run or restart Init can fail that attempt. |
| One child fails six times within five minutes | Its group stops all children, last started first, and passes the failure to its parent. |
| A nested group passes a failure to its parent | The parent restarts that group under the same restart rule. |
| A failure reaches the root | The program panics with the last failure value. |

The delay is one second after the first failure. It doubles for each restart in
the last five minutes. Borrow continues to select the last published value during
a delay and after stop. A call in progress finishes on its selected value.

## Child groups

| Parent | Lifetime |
| --- | --- |
| `root := supervise.NewRoot(report)` | Application group, until root.Stop. Pass nil to disable reports. |
| `group := supervise.StartGroup(parent)` | Nested group; services stop together and restart together after a child exceeds its limit. |
| `s.Children()` | Children of one service run; a restart creates a new group. |

Create groups with these APIs. A zero Group cannot run. Groups otherwise follow
ordinary Go rules. A stopped parent supplies a stopped nested group.

### Child lifetimes

| Event | Effect |
| --- | --- |
| Parent Init returns an error or panics | Stop children from that setup attempt. Application resources remain Init's responsibility. |
| A method starts a child before parent Run begins | Child Init runs immediately; child Run waits for parent Run. If the parent never runs, the child never runs. |
| Parent is stopped | Stop children last started first and wait for them before cancelling the parent's Run context. |
| Parent Run returns or panics | Stop and wait for children before the run ends or the restart delay begins. |
| A child exceeds its restart limit | Its group stops the children and passes the failure up. A nested group can restart first; failure that reaches the service ends that run. |
| A method starts a child after the parent run ends | Init can succeed and return a reference, but child Run cannot run. A saved group from that run has the same limit. |

Create children and any reservations again in each parent run. An old child Ref
still selects the old child after a parent restart. Keep child references in
parent fields and expose methods that select the current child. The checker
permits Ref copies; it does not prove their lifetime.

## Shutdown

| Operation | Behavior |
| --- | --- |
| `done := root.Stop()` | Request shutdown without waiting. Repeated calls return the same channel. |
| `<-root.Stop()` | Wait until all children stop. |
| `<-group.Stop()` on a nested group | Also wait for removal from the parent. Other groups continue. |

Run must return when its context ends. Stop cannot force Run or Init to return
and does not wait for direct method calls. Release locks that children need
before waiting for Stop. A child or report function must not wait for its own
group to stop: the group is waiting for that code to finish. A child can call a
parent during shutdown, but must not wait for a parent method that waits for
that child's group to stop.

## Failure reports

The report argument to NewRoot has type `func(value any, stack []byte)`:

- It receives each recovered panic or restart Init error before the delay.
- Nested groups use the same function; a failure passed up is reported again.
- Reports for different children can run concurrently.
- It must return. The group waits for it and does not recover its panics.
