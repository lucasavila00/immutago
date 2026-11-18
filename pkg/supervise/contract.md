# Service runtime contract

A service embeds `supervise.Service[A]`. The runtime calls Init and Run.
Other method calls run in the caller's goroutine and can overlap with Run and
other calls. See the [service guide](../../docs/guides/services.md) for an example
and the [API](readme.md) for signatures.

| Item | Requirement |
| --- | --- |
| Service declaration | Use the [value embedding forms](../../docs/reference/service-results.md#service-embedding--exhaustive). |
| Setup | Declare `func (*Worker) Init(*Supervisor) error`, promote a declared pointer method, or use the default Init from Service[A]. |
| Background work | Declare `func (*Worker) Run(context.Context)`, or promote a declared pointer method. See the complete [receiver forms](../../docs/reference/service-results.md#lifecycle-receivers--exhaustive). |
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

Start calls Init and Run on the service it creates. Their declared pointer
receivers prevent an implicit receiver copy: Init changes the original service,
and Run can lock shared fields before reading them. A value Run would copy fields
in the watcher after publication, before its body could take a lock. The immutago
analyzer rejects value and interface receivers for these two methods.

A service copy does not start another Run or extend the original service's
lifetime. Other value methods follow the same field rules as explicit copies.
im rules still apply to protected fields.

Standard `go vet ./...` reports common mutex and Supervisor copies. It misses
copies made by dereferencing a call result. These copies still need to obey the
copy rules above.

| Copy form | Standard vet check |
| --- | --- |
| `copy := *ref.Borrow()` | No copy diagnostic, even when the service contains a mutex or Supervisor by value. |
| `p := ref.Borrow(); copy := *p` | Reports a copy when the service contains a mutex or Supervisor by value. |
| `copy := *supervise.NewRoot(nil)` or `copy := *supervise.StartSupervisor(parent)` | No copy diagnostic. Keep the returned pointer. |

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
| Terminal panic | It runs in a runtime goroutine after accepted Init calls, reporting, and cleanup. A caller cannot recover it. A root from NewRootNoPanic does not panic. |

Each accepted Start creates one service value and calls Run once after successful Init.
Borrow returns that same value, including after it stops.

## Supervisors

| Parent | Lifetime |
| --- | --- |
| `root := supervise.NewRoot(report)` | Application supervisor, until root.Stop or root.Shutdown. Pass nil to write failure reports and their original stacks to standard error. |
| `root := supervise.NewRootNoPanic(report)` | The same supervisor without the terminal panic. See [Keep the process after a failure](#keep-the-process-after-a-failure). |
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
| Stop or Shutdown on a nil or zero Supervisor | Panic with `supervise: nil or zero Supervisor`. |
| `<-root.Stop()` | Wait for accepted Init calls and all children to stop. A terminal failure prevents normal completion. A root from NewRootNoPanic closes the channel after a failure; only Shutdown returns that failure. |
| `err := root.Shutdown(ctx)` | Call Stop, then wait as `<-root.Stop()` does until ctx ends. See [Limit the wait](#limit-the-wait). |
| `<-supervisor.Stop()` on a nested supervisor | Wait for this supervisor's children and removal from its parent. This means the subtree has ended; a child failure still stops the root. |
| Stop is requested | Close admission immediately. Each supervisor waits for its accepted Init calls, then cancels children in reverse registration order. |
| A registered service has not been admitted to Run when Stop is requested | Wait for its turn, stop its children, then call Run with an already-cancelled context. |

Cleanup panics and Goexit are terminal failures.
Services register after Init; concurrent Start calls can register in a different
order from their invocation order. Nested supervisors register when StartSupervisor is called.

Stop does not interrupt work already admitted. Registered work must finish
before the completion channel closes. Run must return when its context ends.
Stop and Shutdown cannot force Run or Init to return
and do not wait for direct method calls. Release locks that children need
before waiting for Stop. A service must not wait from Run for the supervisor
that contains it: that supervisor is waiting for Run to finish. The supervisor
passed to Init contains only the services it starts; it can stop and wait for
that supervisor. A report function must not wait for tree shutdown. A child can call a
parent during shutdown, but must not wait for a parent method that waits for
that child's supervisor to stop. These rules for a wait on Stop also apply to Shutdown.

### Limit the wait

`err := s.Shutdown(ctx)` calls `s.Stop()`. Then it waits for the channel that
Stop returns or for the end of ctx:

| Condition | Result |
| --- | --- |
| The supervisor stops before ctx ends | Return nil, except for a root after a failure. |
| The supervisor has stopped and ctx has ended | Return nil, except for a root after a failure. |
| ctx ends first | Return a `*supervise.ShutdownError` that wraps `ctx.Err()`. Do not cancel or force work. Ordered shutdown continues. A later Stop or Shutdown waits for the same shutdown. |
| ctx has ended before the call and the supervisor has not stopped | Call Stop, which starts shutdown. Then return the same error type. |
| Concurrent Stop and Shutdown calls | All calls use one shutdown. |
| Nested or service-owned supervisor | Use the same rules for its subtree. There is no terminal panic. |
| Root after a failure | Do not return nil. When ordered shutdown completes, the terminal panic ends the process. Before that, return the error for the end of ctx. |
| Root from NewRootNoPanic after a failure | Return the first `*supervise.Failure` when ordered shutdown completes. Before that, return the error for the end of ctx. |

Use `errors.As` with `*supervise.ShutdownError` to identify the end of ctx, and with
`*supervise.Failure` to identify a failure. Do not use `errors.Is` with a context error
for this: a Failure wraps the panic value, which can also be a context error.
The Waiting field of ShutdownError names the wait that was in progress:

| Waiting | Meaning |
| --- | --- |
| Go type string of a service, for example `main.Worker` | This service had not finished its Run or its failure report. Shutdown follows nested and service-owned supervisors to the service in progress. |
| `accepted Init calls` | A supervisor waited for an Init that Start had accepted. |
| Empty | No wait was in progress when ctx ended, for example between two children. |

Shutdown stops one child at a time. A service that registered before the named service
has not received cancellation yet.

Without a limit, the wait continues while a Run ignores its context, an accepted
Init blocks, or a report blocks. If this occurs after a root failure, the
terminal panic does not occur and the process stays alive with admission closed.
After a root failure, Shutdown does not return nil. Use the ctx to limit the
wait, then end the process:

```go
failed := make(chan struct{}, 1)
root := supervise.NewRoot(func(value any, stack []byte) {
	// log value and stack
	select {
	case failed <- struct{}{}:
	default:
	}
})
// start services
select {
case <-signals.Done():
case <-failed:
}
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
if err := root.Shutdown(ctx); err != nil {
	os.Exit(1)
}
```

In this example, signals is the application's signal context. The report must
return. The send to failed does not block, because a full channel selects the
default case. After a failure, the process ends in one of two ways: the terminal
panic when ordered shutdown completes, or the application's exit when ctx ends first.

### Keep the process after a failure

After a failure, a root from NewRoot ends the process with the terminal panic when
ordered cleanup of the tree completes. Application cleanup outside the tree, such
as a flush of buffered data, cannot complete after that panic.

A root from NewRootNoPanic reports the failure and stops the tree in the same way.
It does not panic. The channel from Stop closes, and Shutdown returns the first
`*supervise.Failure`. The application does its remaining cleanup and then ends the process:

```go
failed := make(chan struct{}, 1)
root := supervise.NewRootNoPanic(func(value any, stack []byte) {
	// log value and stack
	select {
	case failed <- struct{}{}:
	default:
	}
})
// start services
select {
case <-signals.Done():
case <-failed:
}
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
err := root.Shutdown(ctx)
// flush data and close resources that the tree does not own
if err != nil {
	os.Exit(1)
}
```

The application must end the process. A stopped root does not start services again.

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
