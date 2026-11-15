# Service checks and runtime

## Result checks

[Service classification](../../internal/checker/types.go) follows embedded fields
to the resolved `supervise.Service` type, including aliases and pointers.
[Result checking](../../pkg/analyzer/service.go) visits the pointer method set:

1. Skip unexported methods, Init, Run, and methods supplied by the runtime.
2. Accept no result or one result, including for promoted methods.
3. Match the [service result types](../service-results.md#result-types--exhaustive).
   Prove scalars and ByValue structs with the existing immutable type proof;
   check each unnamed array's element type recursively. Fields of a ByValue
   struct use the ordinary stored-type rules.

Start arguments, method parameters, callbacks, and pointers from Current follow
ordinary Go rules. Protected data passes through the im type proofs, source
matcher, and Freeze checks.

The classifier treats `Service[A]` as runtime storage, so a service can receive
im values through its start arguments. When A contains im data, its stored-value
proof must pass; this prevents retaining a ByValue pointer. Ordinary A types need
no immutable proof. `Ref[T]` is a mutable service reference, not an allowed
immutable stored type. Runtime calls use the fixed library operations.

The source matcher accepts the fixed Start result forms when arguments contain
protected storage, including two local result bindings and a matching named
function return. Calls with only ordinary arguments use ordinary Go binding
rules. The `Init` error result does not permit `error` in other exported service
method signatures. [CLI fixtures](../../tests/testdata/accepted/serviceinit/main.go)
and [runtime tests](../../pkg/supervise/init_test.go) check these rules.

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
