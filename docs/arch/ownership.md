# Object and builder storage

## Immutable objects

| Rule | Implementation |
| --- | --- |
| `im.Shared` and `im.Scoped` fields cannot change after construction | [Write matcher](../../pkg/analyzer/subset_writes.go): allow whole-variable and tracked builder-element replacement; reject protected field, array-element, and dereference writes. |
| Protected pointers start at fresh allocations | [Source matcher](../../pkg/analyzer/subset_operations.go): reject `&variable`, which could expose storage that is later replaced. |
| Shared pointers can be stored; Scoped pointers cannot | Source matcher checks uses; [type proof](../../internal/checker/proof.go) checks stored pointer targets. |

For [im.UncheckedValue](../../pkg/im/unchecked_value.go):

- The type proof accepts the handle without inspecting its payload.
- A handle field in a marked object is still read-only.
- [Trust and Unwrap](../../pkg/analyzer/unchecked_value.go) require the package
  that declares T, resolved through aliases and function values.
- The handle has no methods that could expose its pointer.
- The declaring package supplies the immutability promise.

## Type markers

One marker sets the rule for every use of a type. Callers do not repeat it on
fields, parameters, receivers, or local names:

```go
type Config struct {
    im.Scoped
    Limit int
}

func limit(config *Config) int { return config.Limit }
```

Every function that receives `*Config` gets the Scoped rule, regardless of the
parameter's name.

The marker selects the pointer rule:

- `im.Shared`: callers can store or return pointers.
- `im.Scoped`: callers can read through pointers, but cannot store them in fields
  or globals, capture them, or return them.

For either marker, the checker requires:

- Immutable fields, including nested fields and array elements.
- Immutable targets for stored pointers, so another pointer cannot change them.
- Allowed key and element types in frozen collections.
- No writes to those fields through another pointer or a receiver method.

One type has one pointer policy. Owner requires a Scoped type. Use separate
marked types or value copies when callers need different pointer policies.
The checker proves the fields and checks each operation.

See [Shared](../shared.md), [Scoped](../scoped.md), and [Owner](../owner.md) for
the supported forms.

## Atomic Scoped slots

- [im.Owner](../../pkg/im/owner.go) is a mutable `atomic.Pointer` slot, allowed
  as a package variable or a direct named field of an unmarked struct.
- The [slot matcher](../../pkg/analyzer/owner_slot.go) checks the type argument,
  receiver, direct call, fresh Store input, and local Load binding.
- The holder check rejects copies of slots and their containing structs. It
  also checks holder pointer uses. A closure can access a slot; a loaded pointer
  still has the Scoped capture restriction.
- Loaded pointers follow Scoped rules. Store replaces the slot's pointer;
  previous objects stay immutable. Go's GC manages their lifetime.

## Services and Scoped results

[Service classification](../../internal/checker/types.go) follows embedded fields
to the resolved `supervise.Service` type, including aliases and pointers.
[Result checking](../../pkg/analyzer/service.go) visits the pointer method sets
of named and anonymous service structs:

1. Skip unexported methods, Init, Run, and methods from the fixed runtime.
2. Accept no result or one result, including for promoted user methods.
3. Use `Prover.ScopedResultReason` to match primitives, unnamed arrays of Scoped
   results, service handles, and structs with im.Scoped.
4. Check each array element type. Check each Scoped field with the ordinary
   stored-type rules. For `Ref[T]`, require a concrete service struct as T.

The [Scoped-result reference](../scoped-results.md) lists the allowed types.
Start arguments and method inputs follow ordinary Go rules. When `Service[A]`
stores im data, A must pass the stored-type checks. This permits Scoped values
but rejects Scoped pointers. Ordinary A types need no immutable proof.
All im operations keep their type, source-form, and Freeze checks.

### Service-handle boundary

The checker restricts calls through `Ref`. It does not check how a service
method uses ordinary mutable data.

| Checked code | Required check | Source |
| --- | --- | --- |
| `Ref[T]` | T is a concrete struct that embeds `supervise.Service`. The check does not inspect T's mutable fields. | [refTargetReason](../../internal/checker/proof.go) |
| `ref.Current().Message(args)` | Current's result is the receiver of this direct call. Message is exported, is a user method, and is not Init or Run. Promoted messages are allowed; promoted Current calls are rejected. | [matchCurrent](../../pkg/analyzer/service_handles.go) |
| Message result | No result or one Scoped result, including for promoted user methods. | [checkServiceResults](../../pkg/analyzer/service.go) |
| Handle copies, storage, and calls | Match the [allowed forms](../service-handles.md#handle-operations--exhaustive). Reject conversions, interface assignments, type assertions, and type-switch recovery. A producer interface must keep the same interface type. | [matchHandleUse](../../pkg/analyzer/service_handle_uses.go) |
| User generic type arguments | Reject handles, including inside fields or callback types. Concrete Ref parameters, fields, and results are allowed. | [handleInstance](../../pkg/analyzer/service_handles.go) |
| Functions with handles in receivers, parameters, or results | Require a Go body. This includes handles inside fields and callback types. | [checkFunctionBody](../../pkg/analyzer/analyzer.go) |
| Handle fields in marked objects | Reject field writes after construction. | [matchWrite](../../pkg/analyzer/subset_writes.go) |

These checks use compiler-resolved types. [Classifier.Handles](../../internal/checker/classification.go)
finds handles inside pointers, fields, native containers, function signatures,
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

Visit is exported and returns no result, so the handle check accepts the call.
Visit gives its receiver to the callback, which changes count. The same method
can also save its receiver in a global. Review those operations and the locks
in the service code. The checker does not prove that service state stays private
or that concurrent calls are safe.

The checks apply to [selected packages](../rules.md#package-boundary).
The [trusted code list](validation.md#trust-boundary) still applies.

### Publication and child lifetimes

| Operation | Runtime action |
| --- | --- |
| Copy Ref | Copy a pointer to an atomic pointer; all copies select the same current value. |
| StartReserved | Create a fresh service, set arguments, call Init, then publish with compare-and-swap. A losing concurrent start stops its staged children before panicking. |
| Restart | Repeat setup; publish with an atomic store only after success. Failure leaves the previous value available. |
| Current | One atomic load, no allocation. |
| Children created during Init | Register in that run's child group; wait until Run begins. Failed setup stops them. |
| Shutdown | Stop the child group before cancelling the parent's Run context. |
| Run returns or panics | Finish child cleanup before ending or waiting to restart. |
| Start a child after the run ends | Init can run, but child Run cannot: the child group is stopped. |

The checker does not establish whether a saved child Ref belongs to the current parent run.

The source matcher accepts the fixed Start result forms when arguments contain
protected storage, including two local result bindings and a matching named
function return. Calls with only ordinary arguments retain ordinary Go binding
rules. The `Init` error result does not admit `error` into other exported service
method signatures. [CLI fixtures](../../tests/testdata/accepted/serviceinit/main.go)
and [runtime tests](../../pkg/supervise/init_test.go) check these separate rules.

## Native builders

| Stage | Work |
| --- | --- |
| [Discovery](../../internal/freeze/discover.go) | Index local declarations by compiler variable identity. Track containers with protected contents and native Freeze inputs. |
| Initial source | Match allocation, literal, bare nil, zero declaration, or exact collection Clone. |
| [Use checks](../../internal/freeze/uses.go) | Reject handle copies, subslices, slot addresses, escapes, and unmatched calls. |

Copying an immutable element pointer does not alias a builder slot; taking the
slot's address does. Follow selectors and indices only until they produce a separate value.

[Flow analysis](../../internal/freeze/flow.go) expands each participating
function's Go control-flow graph into ordered use, consume, and reset events.
A forward solver tracks sets of consumed builder identities in bitsets. Each
function assigns dense IDs to its builders and retains one input and output
bitset per event:

- A join takes the union of predecessor states.
- Freeze adds the input identity after argument evaluation.
- A fresh declaration clears its identity, including on each loop iteration.
- A later use fails if any incoming path has consumed the builder.

The solver visits reachable points in reverse postorder until sets stop
changing. Delayed writes are recorded after operand evaluation. Freeze inside
an active range over the same builder is rejected because the range retains
the handle. [Source attribution](../../third_party/rustc/README.md) covers the
graph traversal.

## Runtime representation

- [FrozenSlice](../../pkg/im/slice.go) and [FrozenMap](../../pkg/im/map.go) hold private native handles.
- Freeze copies the handle; the analyzer enforces exclusive use and consumption.
- Reads and [iteration](../../pkg/im/iterator.go) use the native storage directly.
- Clone copies slots, not referenced objects. Those objects remain safe to share because they are immutable.
