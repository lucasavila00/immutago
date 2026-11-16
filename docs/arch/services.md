# Service runtime and checks

`supervise` uses one watcher per service and one shared atomic slot per Ref.
Method calls run in the caller's goroutine.

Service, TerminalService, and Group keep mutable ownership state in their values. They must not be
copied after initialization. Each contains a zero-size `noCopy` field for standard
vet's copylocks check. It adds no lock or runtime check. Immutago does not run that
check; callers must also use standard `go vet`. See the
[copy contract](../reference/services.md#copy-contract).
Copylocks is a diagnostic check, not a complete proof that no value was copied;
for example, it does not inspect channel sends.

## Initialization, publication, and calls

[Ref[T]](../../pkg/supervise/supervise.go) contains `*atomic.Pointer[T]`:

```text
Ref copy ─┐
Ref copy ─┴──> shared slot ──> latest initialized *T
                    ▲
             successful retry publishes here
```

| Stage | Execution owner | Action |
| --- | --- | --- |
| Initial setup | Start caller | Allocate T; bind arguments and parent group; call Init. |
| Initial Init failure | Start caller | Stop staged children. Return a zero Ref and error, or propagate the panic. |
| Initial publication | Start caller | Store initialized T; register a watcher with the group. |
| Run admission | Watcher, under the tree mutex | Check policy, activate staged child groups, and publish a replacement as one decision ordered against Stop. |
| Background work | Watcher, outside the mutex | Call Run once after successful Init. |
| Stopped or pending initialized attempt | Watcher | Wait for its entry's shutdown turn; stop children, then call Run with an already-cancelled context for cleanup. |
| `ref.Borrow().Method(args)` | Method caller | Load the slot; call the method directly on that T. |
| Replacement setup | Same watcher | Check policy under the tree mutex; allocate a fresh T and call Init outside the mutex. Successful Init still needs Run admission. |

| Slot change | Effect on callers |
| --- | --- |
| Retry Init fails | The previous T remains selected. |
| Retry publishes a new T | Later Borrow calls select it; existing calls can finish on the old T. |
| Group stops | The slot keeps its last value. Stop does not wait for direct method calls. |

## One watcher owns retries

Service and TerminalService embed the same private service state. Their sealed
bind methods return that state and a fixed retry limit: five for Service, zero for
TerminalService. Start keeps the service value and that state in one current-attempt record.
Cleanup clears both pointers after the attempt ends.
The same watcher applies either limit. It retains restart timestamps within five
minutes; the delay is `1s << recentRestartCount`.
It records and reports the failure before child cleanup can wait. The retry
window uses the failure time, so cleanup does not move an old failure forward.

| Attempt outcome | Cleanup | Next action |
| --- | --- | --- |
| Run returns | Stop and wait for its children. | Remove the service entry. |
| Service Run panics within its budget | Stop and wait for its children. | Delay, then initialize a replacement. |
| Retry Init fails | Stop children created by that setup. | Retry within the same budget. |
| TerminalService Run panics, budget exhausted, or `runtime.Goexit` | Select and report one terminal Failure before waiting for root shutdown. | After tree cleanup, panic with that Failure. |
| Shutdown begins | Cancel retry policy and pending delays. | Start no further attempt. |

Initial Init precedes watcher registration, so group shutdown cannot wait for
that setup. Retry Init belongs to the registered watcher and is included in its lifetime.
Successful setup always reaches Run. If a stopped group cannot register the first
attempt, Start waits for its cleanup watcher. A successful replacement after Stop
runs cleanup without replacing the published reference.

## Groups and shutdown order

Each service attempt owns a separate child group:

```text
Init: children pending ──> Run admission: activate ──> attempt ends: stop and wait
                     └──> shutdown: cancelled Run for cleanup
next attempt: new child group
```

| Group state | Purpose |
| --- | --- |
| Shared tree mutex | Protect admission, publication, child links, activation, service attempt state, and Stop across all descendant groups. |
| Doubly linked child list | Preserve start order; unlink a completed child without searching. |
| Policy context | Disable retries throughout the subtree immediately. |
| Separate entry contexts | Keep reverse shutdown order, including attempts that finish Init after policy stops. Each service stops its children before it cancels its Run context. |

Shutdown follows this order:

1. Close admission and cancel policy under the tree mutex. Start one shutdown worker.
2. Select the last child under the mutex, then release the mutex.
3. Start a pending child's watcher if needed, cancel that entry, and wait.
   A service's children stop before its Run context is cancelled or its cleanup Run begins.
4. Remove the completed child and repeat until the list is empty.

Only ordered shutdown cancels registered entries. A rejected, unregistered start
is cancelled by the add path. Launch does not read policy or cancel an entry.
No user code, reporting, or cleanup waits run under the tree mutex. Run can enter
its function after Stop if its admission decision came first. Root Stop and
terminal failure use the same mutex; groups do not add retry budgets.
The shutdown worker reports a selected Failure before its first cleanup wait.
It uses the existing callback, or standard error when the callback is nil.

## Analyzer boundary

| Source | Compiler-resolved check |
| --- | --- |
| `Service[A]` or `TerminalService[A]` embedding | The selected runtime embedding path uses value fields. |
| `Ref[T]` | T is a concrete service struct; application fields use Go rules. |
| Direct `Borrow` use | Match the immediate exported method call; exclude lifecycle and runtime methods. |
| Exported service methods | Check each result with the shared ReadResult proof, or compiler identity with predeclared error. Apply no result count or order limit. |
| Retained start arguments | Resolve aliases to the runtime base and prove its stored argument with Frozen when it contains protected data or callbacks carrying it; ordinary arguments use Go contracts. |

[Reference matching](../../pkg/analyzer/service_handles.go) and
[result checks](../../pkg/analyzer/service.go) do not prove that the receiver stays private:

```go
func (c *Counter) Visit(f func(*Counter)) { f(c) }
```

This method passes result checks; its callback can retain `c`. Review locks,
state changes, and input use in service code. The
[service reference](../reference/services.md) defines the complete public forms.
