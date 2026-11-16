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
| Stopped or pending initialized attempt | Watcher | Wait for its entry's shutdown turn; stop children, then call Run with an already-cancelled context for cleanup. |
| `worker := ref.Borrow()` | Method caller | Load the slot once. Calls through worker use that instance. |
| Replacement setup | Same watcher | Allocate a fresh T with the same arguments; publish only after successful Init. |

| Slot change | Effect on callers |
| --- | --- |
| Retry Init fails | The previous T remains selected. |
| Retry publishes a new T | Later Borrow calls select it; existing local pointers still select the old T. |
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
| Separate entry contexts | Keep reverse shutdown order, including attempts that finish Init after policy stops. Each service stops its children before it cancels its Run context. |

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
| `*T` outside T's declaring package | Classify it as Local. Reuse binding, transport, capture, and return checks. Permit only direct exported service methods; exclude lifecycle/runtime methods and field reads. |
| `*T` inside T's package; service values and Ref values | Ordinary Go forms. Im fields keep their field and write rules. |
| Exported service methods | Check each result with the shared ReadResult proof, or compiler identity with predeclared error. Apply no result count or order limit. |
| Retained start arguments | Protected data or callbacks carrying it must pass Frozen; ordinary arguments use Go contracts. |

| Implementation | Responsibility |
| --- | --- |
| [Reference checks](../../pkg/analyzer/service_handles.go) | Validate Ref targets. |
| [Source classifier](../../internal/checker/classification.go) | Keep one fixed package context; classify external service pointers as Local. |
| Field classifiers | Keep im field proofs context-free. Reject ordinary carriers of Local service pointers, including imported and anonymous results. Keep service values and Ref representations ordinary. |
| Source matcher | Check actual pointer values by resolved type, including factory, interface, and generic call results. |
| Producer contracts | Keep service-pointer producers ordinary. Restrict Local-pointer inputs and im.Local outputs with the common call rules. |
| Service implementation | Synchronize mutable state and review input use. The checker does not prove isolation or synchronization. |

See the complete [service reference forms](../reference/service-references.md).
