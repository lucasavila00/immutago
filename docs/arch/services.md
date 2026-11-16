# Service runtime and checks

`supervise` uses one watcher per service and one shared atomic slot per Ref.
Method calls run in the caller's goroutine.

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
| Background work | Watcher | Activate the service's child group; call Run once after successful Init. |
| Stopped or pending initialized attempt | Watcher | Stop children, then call Run with an already-cancelled context for cleanup. |
| `ref.Borrow().Method(args)` | Method caller | Load the slot; call the method directly on that T. |
| Replacement setup | Same watcher | Allocate a fresh T with the same arguments; publish only after successful Init. |

| Slot change | Effect on callers |
| --- | --- |
| Retry Init fails | The previous T remains selected. |
| Retry publishes a new T | Later Borrow calls select it; existing calls can finish on the old T. |
| Group stops | The slot keeps its last value. Stop does not wait for direct method calls. |

## One watcher owns retries

The watcher retains restart timestamps within five minutes. It permits five
restarts; the delay is `1s << recentRestartCount`.
It records and reports the failure before child cleanup can wait. The retry
window uses the failure time, so cleanup does not move an old failure forward.

| Attempt outcome | Cleanup | Next action |
| --- | --- | --- |
| Run returns | Stop and wait for its children. | Remove the service entry. |
| Run panics | Stop and wait for its children. | Delay, then initialize a replacement. |
| Retry Init fails | Stop children created by that setup. | Retry within the same budget. |
| Budget exhausted or `runtime.Goexit` | Select and report one terminal Failure before waiting for root shutdown. | After tree cleanup, panic with that Failure. |
| Shutdown begins | Cancel retry policy and pending delays. | Start no further attempt. |

Initial Init precedes watcher registration, so group shutdown cannot wait for
that setup. Retry Init belongs to the registered watcher and is included in its lifetime.
Successful setup always reaches Run. If a stopped group cannot register the first
attempt, Start waits for its cleanup watcher. A successful replacement after Stop
runs cleanup without replacing the published reference.

## Groups and shutdown order

Each service attempt owns a separate child group:

```text
Init: children pending ──> parent Run: activate ──> attempt ends: stop and wait
                     └──> shutdown: cancelled Run for cleanup
next attempt: new child group
```

| Group state | Purpose |
| --- | --- |
| Mutex | Protect admission, child links, activation, and the stop decision. |
| Doubly linked child list | Preserve start order; unlink a completed child without searching. |
| Policy context | Disable retries throughout the subtree immediately. |
| Separate Run contexts | Cancel services one at a time in reverse start order. |

Shutdown follows this order:

1. Close admission and cancel policy under the group mutex. Start one shutdown worker.
2. Select the last child under the mutex, then release the mutex.
3. Stop that child and wait. Start a pending child's watcher with a cancelled context.
   A service's children stop before its Run context is cancelled or its cleanup Run begins.
4. Remove the completed child and repeat until the list is empty.

Waiting outside the mutex lets watchers remove their entries. Root Stop and
terminal failure share one decision lock; groups do not add retry budgets.
The shutdown worker reports a selected Failure before its first cleanup wait.
It uses the existing callback, or standard error when the callback is nil.

## Analyzer boundary

| Source | Compiler-resolved check |
| --- | --- |
| `Service[A]` embedding | The selected runtime embedding path uses value fields. |
| `Ref[T]` | T is a concrete service struct; application fields use Go rules. |
| Direct `Borrow` use | Match the immediate exported method call; exclude lifecycle and runtime methods. |
| Exported service methods | Apply the [result signature rule](../reference/service-results.md), using the shared ReadResult proof. |
| Retained start arguments | Protected data or callbacks carrying it must pass Frozen; ordinary arguments use Go contracts. |

[Reference matching](../../pkg/analyzer/service_handles.go) and
[result checks](../../pkg/analyzer/service.go) do not prove that the receiver stays private:

```go
func (c *Counter) Visit(f func(*Counter)) { f(c) }
```

This method passes result checks; its callback can retain `c`. Review locks,
state changes, and input use in service code. The
[service reference](../reference/services.md) defines the complete public forms.
