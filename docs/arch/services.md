# Service checks and runtime

A service is a mutable Go struct with an embedded `supervise.Service[A]` value.
`A` is its start argument type. The struct supplies `Run(context.Context)` and
can supply `Init() error`. A `Group` owns its lifetime. A `Ref[T]` selects the
latest initialized `T` through a shared atomic slot.

These parts solve different problems. The runtime starts, retries, and stops
services. The analyzer checks reference targets, direct Borrow calls, and method
result types. Neither part supplies locks for application fields.

## From Start to a method call

Consider a service that loads data in Init and serves reads through an exported
method. Its caller first uses `Start[T](group, args)`, which returns `(Ref[T],
error)`. It then calls `ref.Borrow().Method(args)`. That method runs in the
caller's goroutine; there is no mailbox or request queue.

[Start](../../pkg/supervise/supervise.go) performs these steps:

1. Allocate a fresh T. Bind its arguments and parent group before Init.
2. Run Init in the caller. If Init fails or panics, stop children created by
   that attempt. Return a zero Ref on error; let an initial panic reach the caller.
3. Allocate the shared atomic slot and publish the initialized T.
4. Register a watcher with the parent group. An active group launches it; a
   pending group keeps it for later activation. If shutdown has begun, clean
   the initialized value without calling Run.
5. The watcher calls Run with the service's run context. It cleans the attempt
   and its children when Run ends.

Execution stays with these owners:

| Work | Execution owner |
| --- | --- |
| Initial Init | The Start caller. It has not registered a watcher yet. |
| Run, its deferred cleanup, retry Init, and retry delay | The registered service watcher goroutine. |
| A method called through Borrow | The calling goroutine. The watcher does not serialize it. |
| Ordered group shutdown | One shutdown goroutine per stopped group. It waits outside the group mutex. |

A copy of Ref copies a pointer to the slot. Borrow performs one atomic load of
that slot and returns `*T`. A retry publishes a new initialized T in the same
slot. A call already in progress can finish on an older T. Stop does not clear
the slot or wait for direct method calls. `IsZero` only checks whether the slot
pointer is nil; it does not check readiness or shutdown.

## What the analyzer proves

The analyzer uses compiler type identity, not field or method spelling alone.
The [service classifier](../../internal/checker/types.go) finds the embedded
runtime Service through its selected bind method. Each field on that selected
embedding path must be a value. This prevents a fresh service from having a nil
embedded runtime pointer. Other pointer fields in the service use Go rules.

A `Ref[T]` proof requires a concrete service struct T and the valid embedding
path. It does not inspect T's mutable application fields. A Ref can be copied,
stored, passed through an interface, or used in a generic function under Go
rules. Its copies still select the same slot.

[The Borrow matcher](../../pkg/analyzer/service_handles.go) checks each direct
use of the runtime Borrow selector. The returned pointer must be the receiver
of an immediate exported service method call. Init, Run, and runtime methods
are excluded. Promoted service methods can be called; promoted Borrow selectors
cannot. Saving `ref.Borrow()`, saving Borrow as a function, or saving a bound
service method fails this match.

[Result checking](../../pkg/analyzer/service.go) visits the pointer method set,
including promoted methods. Each exported user method other than Init and Run
must return no result, one ReadResult, `error`, or `(ReadResult, error)`.
ReadResult admits booleans, numbers, Copied records, service references, Local
records and projections, unchecked handles, and unnamed arrays of read results.
A Frozen record by itself is not a ReadResult: return its local projection for
read access. Error means the predeclared Go interface, with aliases accepted.
The checker does not inspect error origins or payloads.

Start arguments are retained for retries. When `Service[A]` contains protected
im data or a callback carrying that data, A must pass the Frozen proof. This
excludes retained Local values. Ordinary arguments and method parameters use
Go contracts. Protected im arguments and results also pass the normal source,
write, and transport checks; the runtime API does not bypass them.

### What remains a service review

The direct Borrow rule does not prove that the service pointer stays private.
For example, this exported method can pass its receiver to a callback:

```go
func (c *Counter) Visit(call func(*Counter)) { call(c) }

func change(ref supervise.Ref[Counter]) {
    ref.Borrow().Visit(func(c *Counter) { c.count++ })
}
```

The method returns no result and the direct call has the required form. Its
body can also save the receiver in a global. Indirect interface or generic
calls can expose it as well. Review state changes, locks, callback contracts,
and input use in the service code. `ConcurrentSafe[T]` records a caller's
promise; it adds no synchronization or analyzer rule.

## One attempt owns one child group

`Service.Children()` creates a group for that service value. During Init the
group is pending: child Init calls can finish, but child Run calls wait. When
the parent Run begins, the runtime activates its children. Failed Init stops
the pending children. Every retry creates a new service and a new child group.

On ordered shutdown, the runtime stops and waits for the child group before it
cancels the parent's Run context. When Run returns or panics, attempt cleanup
also closes child admission and waits for child cleanup before a retry. A later
Children call on the ended value returns a stopped group. A saved child Ref can
still select its last value; the checker does not tie it to the current parent
attempt.

## Group registration and shutdown

A Group stores children in start order in a doubly linked list. Each child has
its own previous and next links. The group mutex protects admission, list
changes, activation, and the decision to stop. Removal clears both links.
This permits removal of a completed child without a list search.

Groups activate at most once and stop at most once. A root starts active; a
service's child group starts pending. `StartGroup` adds a nested lifetime group,
not another retry budget. Stop closes admission and cancels the group policy
context. One shutdown goroutine then selects the last child under the mutex,
releases the mutex, and waits for that child to stop. It repeats until the list
is empty. Waiting outside the mutex lets completed watchers remove themselves.
No new entry can be added after admission closes.

The policy context prevents new retries across the subtree at once. Each Run
has a separate context, cancelled in reverse start order. Thus older services
can remain available while newer services stop. Initial Init runs before
registration, so Stop cannot wait for that unregistered setup. Restart Init
belongs to an existing watcher and is part of its lifetime. A service must not
wait for its own group to stop from Run: the group is waiting for Run to end.

## Retry and terminal failure

One watcher owns one service's retry history. A normal Run return ends the
entry. A panic, or an Init error during a retry, can start a new attempt after
cleanup. The watcher keeps restart times within five minutes. It permits five
restarts in that window, with delays of 1, 2, 4, 8, and 16 seconds when no prior
restart expires. Shutdown interrupts the delay. A failed replacement Init leaves
the previous value in the Ref slot.

An exhausted failure goes directly to the root. Groups and ancestor services
do not retry it. The root uses the same lock for Stop and terminal failure,
selects one terminal outcome, stops the tree, then panics with `*Failure`.
`runtime.Goexit` is terminal; a defer in the watcher detects it and reports it.
The optional report callback receives failures and stacks. Calls can overlap
and must return without waiting for their group to stop.

[Runtime tests](../../pkg/supervise/supervise_test.go) check retry and stop
behavior. [Init tests](../../pkg/supervise/init_test.go) check publication and
failed setup. [Registry tests](../../pkg/supervise/registry_test.go) check list
removal and shutdown order. [Reference tests](../../pkg/analyzer/service_handles_test.go)
check the compiler-resolved Borrow forms. The [service reference](../reference/services.md)
gives the complete supported source forms.
