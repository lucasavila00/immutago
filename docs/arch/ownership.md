# Object and builder storage

[Architecture](README.md) · [Collection costs](../performance.md)

## Immutable objects

Both markers prohibit mutation from construction onward. The
[write matcher](../../pkg/analyzer/subset_writes.go) permits complete variable
replacement and complete tracked native-builder element replacement. It rejects
writes to protected fields, array elements, and dereferenced objects.

Pointers start at fresh allocations of concrete marked structs. Taking a
named variable's address would expose a slot that can later be rebound, so that
form is rejected. `Shared` pointers may be copied, stored, and returned.
`LocalRef` pointers remain local or pass through direct typed calls without
escaping. The [source matcher](../../pkg/analyzer/subset_operations.go) checks
these uses; the [type proof](../../internal/checker/proof.go) rejects stored
`LocalRef` pointers.

## Native builders

[Discovery](../../internal/freeze/discover.go) indexes local declarations by
compiler variable identity. It tracks fresh native containers with protected
contents and native inputs to Freeze. Valid initial sources are allocation,
literal, bare nil, zero declaration, or an exact collection `Clone` call.

[Use checks](../../internal/freeze/uses.go) match the builder operations before
and after Freeze. They reject handle copies, subslices, slot addresses, escape,
and unsupported calls. A copied immutable element pointer does not alias a
builder slot. An address of that slot does. Storage projections are followed
only until an expression produces a separate value.

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

[FrozenSlice](../../pkg/im/slice.go) and [FrozenMap](../../pkg/im/map.go) hold
private native handles. Freeze copies only the handle; it has no runtime alias
or consumed-state checks. Reads and [iteration](../../pkg/im/iterator.go) use
that storage directly. Clone copies the native slots, not referenced objects.
Recursive immutability makes shared object identity safe after this shallow
copy. The analyzer, not the wrappers, enforces the transfer.
