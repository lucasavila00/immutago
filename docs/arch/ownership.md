# Object and builder storage

## Immutable objects

| Rule | Implementation |
| --- | --- |
| `im.Shared` and `im.ByValue` fields cannot change after construction | [Write matcher](../../pkg/analyzer/subset_writes.go): allow whole-variable and tracked builder-element replacement; reject protected field, array-element, and dereference writes. |
| Protected pointers start at fresh allocations | [Source matcher](../../pkg/analyzer/subset_operations.go): reject `&variable`, which could expose storage that is later replaced. |
| Shared pointers can be stored; ByValue pointers cannot | Source matcher checks uses; [type proof](../../internal/checker/proof.go) checks stored pointer targets. |

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
    im.ByValue
    Limit int
}

func limit(config *Config) int { return config.Limit }
```

Every function that receives `*Config` gets the ByValue rule, regardless of the
parameter's name.

The marker selects the pointer rule:

- `im.Shared`: callers can store or return pointers.
- `im.ByValue`: callers can read through pointers, but cannot store them in fields
  or globals, capture them, or return them.

For either marker, the checker requires:

- Immutable fields, including nested fields and array elements.
- Immutable targets for stored pointers, so another pointer cannot change them.
- Allowed key and element types in frozen collections.
- No writes to those fields through another pointer or a receiver method.

One type has one pointer policy. Owner requires a ByValue type. Use separate
marked types or value copies when callers need different pointer policies.
The checker proves the fields and checks each operation.

See [Shared](../shared.md), [ByValue](../by-value.md), and [Owner](../owner.md) for
the supported forms.

## Owner slots

- [im.Owner](../../pkg/im/owner.go) is a mutable `atomic.Pointer` slot, allowed
  as a package variable or a direct named field of an unmarked struct.
- The [slot matcher](../../pkg/analyzer/owner_slot.go) checks the type argument,
  receiver, direct call, fresh Store input, and local Load binding.
- The holder check rejects copies of slots and their containing structs. It
  also checks holder pointer uses. A closure can access a slot; a loaded pointer
  still has the ByValue capture restriction.
- Loaded pointers follow ByValue rules. Store replaces the slot's pointer;
  previous objects stay immutable. Go's garbage collector manages their lifetime.

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
