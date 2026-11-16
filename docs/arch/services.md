# Service runtime and checks

`supervise` uses one watcher and one initialized value per service.
Ref copies hold the same service pointer.
Method calls run in the caller's goroutine.

Service and Group keep mutable ownership state in their values. UncheckedChildren
requires the original service pointer created by Start. The runtime keeps that
pointer through setup, Run, and cleanup. Its tree mutex protects child state.
The zero-size `noCopy` fields enable standard vet diagnostics; they add no runtime
state or check. See the [copy contract](../reference/services.md#copy-contract).

## Initialization, publication, and calls

[Ref[T]](../../pkg/supervise/supervise.go) embeds Frozen and stores its pointer
in UncheckedValue[T]. Start initializes T once
before it creates the reference. Borrow returns that pointer; Stop does not change
it or wait for direct method calls.

| Stage | Execution owner | Action |
| --- | --- | --- |
| Start admission | Start caller, under the tree mutex | Reject closed admission before Init, or count one accepted Start. |
| Setup | Start caller | Allocate T; bind arguments and parent group; call Init once. |
| Init failure | Start caller | Stop staged children before removing the accepted Start count. Return a zero Ref and error, propagate the panic, or continue Goexit. |
| Publication | Start caller | Register the initialized value before removing the accepted Start count. Return Ref even if Stop has since closed admission. |
| Run admission | Watcher, under the tree mutex | Check admission and activate staged child groups as one decision ordered against Stop. |
| Background work | Watcher, outside the mutex | Call Run once after successful Init. |
| Stopped or pending initialized service | Watcher | Wait for its entry's shutdown turn; stop children, then call Run with an already-cancelled context for cleanup. |
| `ref.Borrow().Method(args)` | Method caller | Read the pointer; call the method directly on T. |

## One watcher owns the service run

Run's deferred calls execute before the watcher can observe a failure.
The watcher recovers a Run panic or observes Goexit and captures the original
stack. Under the tree mutex, it records the root's first Failure and requests
shutdown. Stop closes admission; it does not reject failures. Outside the mutex,
the watcher reports one `*Failure`, then stops and waits for its children.

| Run outcome | Action |
| --- | --- |
| Returns | Stop and wait for its children, then remove the service entry. |
| Panics or calls `runtime.Goexit`, including during cleanup | Keep the root's first failure. Report this failure once. Complete ordered shutdown, then panic with the selected Failure. |

Each group has a WaitGroup for accepted Start calls. Admission checks and Add
use the same tree mutex as Stop, so no Add can follow the start of Wait.
The deferred Done follows registration or failed-Init child cleanup. This keeps
staged children owned through their parent's Init, before its watcher exists.

## Groups and shutdown order

Each initialized service owns a child group:

```text
Init: children pending ──> Run admission: activate ──> Run ends: stop and wait
                     └──> shutdown: cancelled Run for cleanup
```

| Group state | Purpose |
| --- | --- |
| Shared tree mutex | Protect admission, child links, activation, service state, and Stop across all descendant groups. |
| Accepted Start count | Wait for Init and failed-Init child cleanup before stopping the group's registered entries. |
| Doubly linked child list | Preserve registration order after Init; unlink a completed child without searching. |
| Admission context | Close admission throughout the subtree immediately. Failure selection does not use this context. |
| Separate entry contexts | Keep reverse shutdown order for admitted and pending services. Each service stops its children before it cancels its Run context. |
| Root failure | Keep the first failure. Every Run belongs to the owned tree, so ordered shutdown waits for accepted Init calls, all reports, and cleanup. |

Shutdown follows this order:

1. Close admission under the tree mutex. Start one shutdown worker.
2. Wait for accepted Start calls. Select the last registered child under the mutex,
   then release the mutex.
3. Start a pending child's watcher if needed, cancel that entry, and wait.
   A service's children stop before its Run context is cancelled or its cleanup Run begins.
4. The watcher removes its entry before it publishes completion. Repeat until the list is empty.
5. Under the mutex, unlink a nested group and close its completion channel, even
   if the root has failed. For the root, read its first failure. With no failure,
   close the channel. With a failure, release the mutex and run the terminal action
   before closing the channel.

Only ordered shutdown cancels registered entries. A rejected Start runs no user
code. Launch does not read admission or cancel an entry.
No user code, reporting, or cleanup waits run under the tree mutex. Run can enter
its function after Stop if its admission decision came first. Root Stop and
terminal failure use the same mutex. This keeps the final shutdown decision and
failure recording in one order, including a failure during ordinary Stop.

The Init barrier applies to each group before cancellation of its registered
entries. Init must not wait for that shutdown or for cancellation that depends
on it. It can stop its own child group to undo setup; the child group does not
wait for its parent Init.
Init must also return if a dependency stops answering. Blocking work needs an
application timeout or independent cancellation. Reporting does not wait for
accepted Init calls, but the terminal panic does.

## Analyzer boundary

| Source | Compiler-resolved check |
| --- | --- |
| `Service[A]` embedding | The selected runtime embedding path uses value fields. |
| `Ref[T]` | Apply the general Frozen wrapper and UncheckedValue field rules. Start constrains the constructed service type. |
| `Borrow` result | Ordinary Go pointer use; retain im checks for protected fields. |
| Exported service methods | Check each result with the shared ReadResult proof, or compiler identity with predeclared error. Apply no result count or order limit. |
| Retained start arguments | Protected data or callbacks carrying it must pass Frozen; ordinary arguments use Go contracts. |

[Type proofs](../../internal/checker/proof.go) and
[result checks](../../pkg/analyzer/service.go) do not prove that the receiver stays private:

```go
func (c *Counter) Visit(f func(*Counter)) { f(c) }
```

This method passes result checks; its callback can retain `c`. Review locks,
state changes, and input use in service code. The
[service reference](../reference/service-references.md) defines the complete public forms.
