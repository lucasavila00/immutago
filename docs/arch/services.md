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
| Setup | Start caller | Allocate T; bind arguments and parent group; call Init once. |
| Init failure | Start caller | Stop staged children. Return a zero Ref and error, or propagate the panic. |
| Publication | Start caller | Create Ref for initialized T; register a watcher with the group. |
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

Init precedes watcher registration, so group shutdown cannot wait for
that setup. Successful setup always reaches Run. If a stopped group cannot
register the service, Start waits for its cleanup watcher.

## Groups and shutdown order

Each initialized service owns a child group:

```text
Init: children pending ──> Run admission: activate ──> Run ends: stop and wait
                     └──> shutdown: cancelled Run for cleanup
```

| Group state | Purpose |
| --- | --- |
| Shared tree mutex | Protect admission, child links, activation, service state, and Stop across all descendant groups. |
| Doubly linked child list | Preserve start order; unlink a completed child without searching. |
| Admission context | Close admission throughout the subtree immediately. Failure selection does not use this context. |
| Separate entry contexts | Keep reverse shutdown order for admitted and pending services. Each service stops its children before it cancels its Run context. |
| Root failure and its entry's completion channel | Keep the first failure and wait for that service, including an unregistered start. Reuse its existing completion channel. |

Shutdown follows this order:

1. Close admission under the tree mutex. Start one shutdown worker.
2. Select the last child under the mutex, then release the mutex.
3. Start a pending child's watcher if needed, cancel that entry, and wait.
   A service's children stop before its Run context is cancelled or its cleanup Run begins.
4. The watcher removes its entry before it publishes completion. Repeat until the list is empty.
5. Under the mutex, read the root failure. With no failure, close the completion channel.
   With a failure, release the mutex, wait for its entry, and run the terminal action
   before closing the channel.

Only ordered shutdown cancels registered entries. A rejected, unregistered start
is cancelled by the add path. Launch does not read admission or cancel an entry.
No user code, reporting, or cleanup waits run under the tree mutex. Run can enter
its function after Stop if its admission decision came first. Root Stop and
terminal failure use the same mutex. This keeps the final shutdown decision and
failure recording in one order, including a failure during ordinary Stop.

| Root state when the first failure is recorded | Terminal action owner |
| --- | --- |
| Shutdown is running or has not started | The shutdown worker waits for the selected entry, then calls the terminal action before closing the root's completion channel. |
| Shutdown is complete | The failed watcher reports, completes its cleanup, and calls the terminal action before its entry finishes. A late Start cannot return first. |

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
