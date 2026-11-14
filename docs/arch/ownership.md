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

## Why the rules belong to the type

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

| Choice | Benefit | Cost |
| --- | --- | --- |
| Type marker | Declare once; match uses with Go type identity. | One type has one pointer policy. A Shared type cannot be used in Owner. Use separate types or value copies when needed. |
| Per-parameter rule | One type could serve both pointer policies. | Repeated annotations and separate policy checks for copies and calls; no stronger immutable-field rule. |

Keep type markers unless a concrete use case shows a large correctness or
simplicity gain. The checker still checks each operation; markers are not promises.

See [Shared](../shared.md), [Scoped](../scoped.md), and [Owner](../owner.md) for
the supported forms.

## Atomic Scoped slots

- [im.Owner](../../pkg/im/owner.go) is a mutable `atomic.Pointer` slot, allowed
  as a package variable or a direct named field of an unmarked struct.
- The [slot matcher](../../pkg/analyzer/owner_slot.go) checks the type argument,
  receiver, direct call, fresh Store input, and local Load binding.
- The holder check rejects copies of slots and their containing structs. It
  also checks holder pointer uses and captures.
- Loaded pointers follow Scoped rules. Store replaces the slot's pointer;
  previous objects stay immutable. Go's GC manages their lifetime.

## Services and groups

- The analyzer matches the [service source forms](../service-forms.md).
- The trusted [runtime](../../pkg/supervise/supervise.go) manages service values and groups.
- Ordinary method bodies can still share mutable data, such as a global map.

### Compiler identities and source forms

[Runtime identities](../../internal/checker/types.go) use the resolved package
path `github.com/lucasavila00/immutago/pkg/supervise` and named objects. Source
spelling and import aliases do not establish identity. The classifier stops at
`Service[A]`; its runtime fields are not immutable application storage. `Ref[T]`
is a leaf of the stored-type proof. The service value itself is not immutable.

The [service matcher](../../pkg/analyzer/service.go) checks the complete shapes
listed in the [service reference](../service-forms.md):

1. A service declaration must be a named, non-generic struct with one embedded
   `Service[A]` value and no immutable marker. `Run` must be declared on its
   pointer receiver with the exact context parameter and no results. `Init` is
   either the embedded default or a declared pointer method with no parameters
   and the exact `error` result. Promoted user entry points do not pass.
2. A service type name is valid only in its declaration, pointer receiver, or
   single type argument of `Start`, `StartReserved`, `Reserve`, or `Ref`.
   Construction outside the runtime therefore fails. A service receiver can
   select fields other than `Service` and make direct synchronous method calls
   other than `Init` and `Run`. Captures and other receiver uses fail.
3. A `Ref.Current()` result must receive a direct synchronous call of an exported
   method declared on that service, other than `Init` and `Run`. The matcher
   checks the resolved method selection and statement context. It rejects
   saved pointers, field reads, promoted calls, method values, and direct go or
   defer calls. The stored-type proof checks each result of the declared exported
   method. Inputs use ordinary Go types; protected input operations still pass
   through the common shape checks.
4. `Children()` must be a direct synchronous call on the service receiver,
   outside a function literal. Its result must be the parent argument of
   `Start`, `StartReserved`, or `StartGroup`, directly or through one new local
   from `:=`. Every use of that local must match the same parent position in
   the same method. A nested group returned by `StartGroup` is ordinary mutable
   data; it can be stored in the service and stopped from another method.
5. The start argument type must pass `Prover.ArgumentReason`. This accepts the
   stored types plus `ConcurrentSafe[T]` as the whole argument or a named field
   of an argument struct with exactly one immutable marker. Service fields
   cannot hold protected data directly, except for the listed Owner forms.

`ConcurrentSafe` has its own [trust contract](../services.md#concurrent-dependencies).
The classifier treats its handle as protected but does not inspect its payload.
The ordinary stored-type proof rejects the wrapper, so it cannot enter an Owner,
a frozen collection, or an exported service method signature. Its source checks
reject defined copies, pointers, arrays, native containers, interface boxing,
and package variables. Exact aliases remain valid. `Value` must be called
directly and synchronously on the receiver's `Args()` result or one field of that
result. `AssumeConcurrentSafe` records a caller promise; it adds no proof or lock.
Owned native builders still need proved key and element types, so ownership does
not admit these argument-only wrappers into collections.

The receiver and reference rules are separate from the marked-object and Owner
holder rules. Each dispatcher path uses the same service classification cache.
Unmatched uses fail instead of falling back to ordinary protected calls.

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
