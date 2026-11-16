# Service runtime and checks

`supervise` uses one watcher and one initialized value per service.
Ref copies hold the same service pointer.
Method calls run in the caller's goroutine.

Service and Group keep mutable ownership state in their values. UnsafeChildren
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
| Initial setup | Start caller | Allocate T; bind arguments and parent group; call Init. |
| Initial Init failure | Start caller | Stop staged children. Return a zero Ref and error, or propagate the panic. |
| Initial publication | Start caller | Create Ref for initialized T; register a watcher with the group. |
| Run admission | Watcher, under the tree mutex | Check admission and activate staged child groups as one decision ordered against Stop. |
| Background work | Watcher, outside the mutex | Call Run once after successful Init. |
| Stopped or pending initialized service | Watcher | Wait for its entry's shutdown turn; stop children, then call Run with an already-cancelled context for cleanup. |
| `ref.Borrow().Method(args)` | Method caller | Read the pointer; call the method directly on T. |

## One watcher owns the service run

The watcher recovers a Run panic or observes Goexit and captures the original
stack. It reports the attempt, then submits a terminal Failure before child
cleanup can wait. The existing tree mutex selects a failure or an earlier Stop.

| Run outcome | Action |
| --- | --- |
| Returns | Stop and wait for its children, then remove the service entry. |
| Panics or calls `runtime.Goexit` | Report the original failure. If Stop did not win, select and report one terminal Failure. Stop the root tree, then panic with that Failure. |
| Fails after Stop is accepted for its subtree | Report the attempt and complete cleanup without a terminal root failure. |

Initial Init precedes watcher registration, so group shutdown cannot wait for
that setup. Successful setup always reaches Run. If a stopped group cannot
register the service, Start waits for its cleanup watcher.

## Groups and shutdown order

Each initialized service owns a child group:

```text
Init: children pending ──> Run admission: activate ──> attempt ends: stop and wait
                     └──> shutdown: cancelled Run for cleanup
```

| Group state | Purpose |
| --- | --- |
| Shared tree mutex | Protect admission, child links, activation, service state, and Stop across all descendant groups. |
| Doubly linked child list | Preserve start order; unlink a completed child without searching. |
| Policy context | Close admission and reject later terminal failures throughout the subtree immediately. |
| Separate entry contexts | Keep reverse shutdown order for admitted and pending services. Each service stops its children before it cancels its Run context. |

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
terminal failure use the same mutex.
The shutdown worker reports a selected Failure before its first cleanup wait.
It uses the existing callback, or standard error when the callback is nil.

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
