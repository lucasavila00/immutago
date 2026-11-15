# Service checks and runtime

## Result checks

[Service classification](../../internal/checker/types.go) follows embedded fields
to the resolved `supervise.Service` type, including aliases and pointers.
[Result checking](../../pkg/analyzer/service.go) visits the pointer method sets
of named and anonymous service structs:

1. Skip unexported methods, Init, Run, and methods supplied by the runtime.
2. Match no result, one read result, `error`, or `(read result, error)` with
   `resultValue`. Identify error by compiler type identity; aliases work.
3. Require the general ReadResult proof for the data result only. It accepts Copied values, Local values, unchecked handles, and unnamed arrays of read results.
   It does not apply another recursive field rule to a returned record.

Start arguments, method parameters, callbacks, and service implementation code
follow ordinary Go rules. Borrow calls use the service reference matcher.
Protected data passes through the im type proofs, source matcher, and Freeze checks.

The classifier treats `Service[A]` as runtime storage, so a service can receive
im values through its start arguments. When A contains im data, its Frozen
proof must pass; this prevents storing Local values or pointers. Ordinary A types need
no immutable proof. `Ref[T]` is an allowed stored type. Its proof checks that T
is a service without inspecting that service's mutable state. Runtime calls
use the fixed library operations.

The source matcher accepts the fixed Start result forms when arguments contain
protected storage, including two local result bindings and a matching named
function return. Calls with only ordinary arguments use ordinary Go binding
rules. The same pair matcher permits direct typed calls returning `(read result, error)`.
It checks each destination and result with the existing write and transport rules.
Errors use ordinary Go contracts; their origins and payloads are not analyzed. [CLI fixtures](../../tests/testdata/accepted/serviceinit/main.go)
and [runtime tests](../../pkg/supervise/init_test.go) check these rules.

## Service reference boundary

The checker restricts calls through `Ref`. It does not check how a service
method uses ordinary mutable data.

| Checked code | Required check | Source |
| --- | --- | --- |
| `Ref[T]` | T is a concrete struct that embeds `supervise.Service`. The check does not inspect T's mutable fields. | [refTargetReason](../../internal/checker/proof.go) |
| `ref.Borrow().Method(args)` | Borrow's result is the receiver of this direct call. Method is exported and is not Init, Run, or a runtime method. Promoted methods are allowed; promoted Borrow calls are rejected. | [matchBorrow](../../pkg/analyzer/service_handles.go) |
| Service method result | The four result signatures, including for promoted methods. | [checkServiceResults](../../pkg/analyzer/service.go) |
| Reference fields in marked objects | Reject field writes after construction. | [matchWrite](../../pkg/analyzer/subset_writes.go) |

These checks use compiler-resolved types. Reference storage and transport follow
ordinary Go rules. There is no service-reference expression allowlist or special
case for atomic storage. The [runtime](../../pkg/supervise/supervise.go) keeps
Ref's fields private. Copies share one atomic slot; Borrow reads that slot once.
An interface or generic method can expose the service pointer outside the
direct-call matcher. This is accepted.

For a fixed runtime method on protected storage, [matchMethod](../../pkg/analyzer/subset_methods.go)
requires a direct call, no promotion, and an exact receiver type from the
instantiated signature. The runtime contract replaces the user marked-receiver
proof. Argument, result, statement, and capture checks still apply. The separate
Borrow matcher runs first. Thus `route.Worker.IsZero()` can copy a reference
from a protected field without permitting saved bound methods or service pointers.

A service can still expose its receiver. For a Counter service with a mutable
count field, this code passes:

```go
func (c *Counter) Visit(call func(*Counter)) { call(c) }

func change(ref supervise.Ref[Counter]) {
    ref.Borrow().Visit(func(c *Counter) { c.count++ })
}
```

Visit is exported and returns no result, so the reference check accepts the call.
Visit gives its receiver to the callback, which changes count. The same method
can also save its receiver in a global. Review those operations and the locks
in the service code. The checker does not prove that service state stays private
or that concurrent calls are safe.

The checks apply to [selected packages](../reference/rules.md#package-boundary).
The [trusted code list](validation.md#trust-boundary) still applies.

## Publication and child lifetimes

| Operation | Runtime action |
| --- | --- |
| Copy Ref | Copy a pointer to an atomic pointer; all copies select the same current service value. |
| IsZero | Compare the reference's pointer with nil; no atomic load or allocation. It does not check service readiness. |
| StartReserved | Create a fresh service, set arguments, call Init, then publish with compare-and-swap. A concurrent start that loses the comparison stops its children before it panics. |
| Restart | Repeat setup; publish with an atomic store only after success. Failure leaves the previous value available. |
| Borrow | One atomic load, no allocation. |
| Children created during Init | Register in that run's child group; wait until Run begins. Failed setup stops them. |
| Shutdown | Stop the child group before cancelling the parent's Run context. |
| Run returns or panics | Finish child cleanup before ending or waiting to restart. |
| Start a child after the run ends | Init can run, but child Run cannot: the child group is stopped. |

The checker does not establish whether a saved child reference belongs to the
current parent run. The [child lifetime reference](../reference/services.md#child-lifetimes)
describes how callers manage these lifetimes.
