# Service checks and runtime

## Result checks

[Service classification](../../internal/checker/types.go) follows embedded fields
to the resolved `supervise.Service` type, including aliases and pointers.
[Result checking](../../pkg/analyzer/service.go) visits the pointer method sets
of named and anonymous service structs:

1. Skip unexported methods, Init, Run, and methods supplied by the runtime.
2. Accept no result or one result, including for promoted methods.
3. Require the general ReadResult proof. It accepts Copied and Local values.
   It does not apply another recursive field rule to a returned record.

Start arguments, method parameters, callbacks, and service implementation code
follow ordinary Go rules. Current calls use the service reference matcher.
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
rules. The `Init` error result does not permit `error` in other exported service
method signatures. [CLI fixtures](../../tests/testdata/accepted/serviceinit/main.go)
and [runtime tests](../../pkg/supervise/init_test.go) check these rules.

## Service reference boundary

The checker restricts calls through `Ref`. It does not check how a service
method uses ordinary mutable data.

| Checked code | Required check | Source |
| --- | --- | --- |
| `Ref[T]` | T is a concrete struct that embeds `supervise.Service`. The check does not inspect T's mutable fields. | [refTargetReason](../../internal/checker/proof.go) |
| `ref.Current().Method(args)` | Current's result is the receiver of this direct call. Method is exported and is not Init, Run, or a runtime method. Promoted methods are allowed; promoted Current calls are rejected. | [matchCurrent](../../pkg/analyzer/service_handles.go) |
| Service method result | No result or one service result, including for promoted methods. | [checkServiceResults](../../pkg/analyzer/service.go) |
| Reference copies, storage, and calls | Match the [allowed forms](../service-references.md#reference-operations--exhaustive). Reject conversions, assignments to a different interface type, type assertions, and type-switch recovery. Interfaces with reference parameters or results must keep the same interface type. | [matchHandleUse](../../pkg/analyzer/service_handle_uses.go) |
| User generic type arguments | Reject service references, including inside fields or callback types. Concrete Ref parameters, fields, and results are allowed. | [handleInstance](../../pkg/analyzer/service_handles.go) |
| Functions with service references in receivers, parameters, or results | Require a Go body. This includes references inside fields and callback types. | [checkFunctionBody](../../pkg/analyzer/analyzer.go) |
| Reference fields in marked objects | Reject field writes after construction. | [matchWrite](../../pkg/analyzer/subset_writes.go) |

These checks use compiler-resolved types. [Classifier.Handles](../../internal/checker/classification.go)
finds service references inside pointers, fields, native containers, function signatures,
and interface methods. It handles recursive types and caches each result.
The [runtime](../../pkg/supervise/supervise.go) keeps Ref's fields private.
Copies share one atomic slot; Current reads that slot once.

A service can still expose its receiver. For a Counter service with a mutable
count field, this code passes:

```go
func (c *Counter) Visit(call func(*Counter)) { call(c) }

func change(ref supervise.Ref[Counter]) {
    ref.Current().Visit(func(c *Counter) { c.count++ })
}
```

Visit is exported and returns no result, so the reference check accepts the call.
Visit gives its receiver to the callback, which changes count. The same method
can also save its receiver in a global. Review those operations and the locks
in the service code. The checker does not prove that service state stays private
or that concurrent calls are safe.

The checks apply to [selected packages](../rules.md#package-boundary).
The [trusted code list](validation.md#trust-boundary) still applies.

## Publication and child lifetimes

| Operation | Runtime action |
| --- | --- |
| Copy Ref | Copy a pointer to an atomic pointer; all copies select the same current service value. |
| StartReserved | Create a fresh service, set arguments, call Init, then publish with compare-and-swap. A concurrent start that loses the comparison stops its children before it panics. |
| Restart | Repeat setup; publish with an atomic store only after success. Failure leaves the previous value available. |
| Current | One atomic load, no allocation. |
| Children created during Init | Register in that run's child group; wait until Run begins. Failed setup stops them. |
| Shutdown | Stop the child group before cancelling the parent's Run context. |
| Run returns or panics | Finish child cleanup before ending or waiting to restart. |
| Start a child after the run ends | Init can run, but child Run cannot: the child group is stopped. |

The checker does not establish whether a saved child reference belongs to the
current parent run. The [child service guide](../service-children.md#child-lifetimes)
describes how callers manage these lifetimes.
