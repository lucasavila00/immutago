# Object and builder storage

## Immutable objects

Both markers prohibit mutation from construction onward. The
[write matcher](../../pkg/analyzer/subset_writes.go) permits complete variable
replacement and complete tracked native-builder element replacement. It rejects
writes to protected fields, array elements, and dereferenced objects.

Pointers start at fresh allocations of concrete marked structs. Taking a
named variable's address would expose a slot that can later be rebound, so that
form is rejected. `Shared` pointers may be copied, stored, and returned.
`Scoped` pointers remain local or pass through direct typed calls that do not
retain them. The [source matcher](../../pkg/analyzer/subset_operations.go) checks
these uses; the [type proof](../../internal/checker/proof.go) rejects stored
`Scoped` pointers.

The type proof accepts [UncheckedValue](../../pkg/im/unchecked_value.go) without
inspecting its payload. The handle is ordinary storage, but its field in a marked
object cannot be replaced. [Trust and Unwrap checks](../../pkg/analyzer/unchecked_value.go) use the
resolved type argument's declaring package, including through aliases and
function values. The handle has no methods, so interface calls and promoted
methods cannot expose its pointer. The declaring package's promise of immutable
data remains unproved.

## Atomic Scoped slots

[Owner](../../pkg/im/owner.go) stores a pointer with `atomic.Pointer`. The slot
is mutable and is not an immutable type. It can be a package variable or a direct
named field of an unmarked struct. The
[slot matcher](../../pkg/analyzer/owner_slot.go) checks its type argument,
receiver, direct call, fresh Store input, and local Load binding. A separate
slot-bearing value check rejects copies of slots and holders. It also checks
holder pointer transport and closure captures. The ordinary Scoped checks
apply to each loaded pointer. A later Store replaces the slot's reference;
earlier loaded objects remain immutable. Go's garbage collector manages lifetime.

## Supervised processes

A [process](../../pkg/proc/proc.go) is mutable and is not a marked struct. The
classifier treats `proc.Process[A]` as opaque, as it treats `im.Owner`, so the
fields of a process are not protected storage. The
[process matcher](../../pkg/analyzer/process.go) applies four rules:

- A process value or pointer is valid only as the receiver variable in a method
  of that process or as a direct `PID.Proc()` result. The receiver can select
  fields other than the marker and call methods directly. A `Proc()` result can
  only receive a direct synchronous call of an exported method other than `Run`
  that is declared on the process.
- The process type name can appear only in its declaration, pointer receivers,
  and the type argument of `proc.Start`, `proc.StartReserved`, `proc.Reserve`, and `proc.PID`.
- Each parameter and result of an exported method other than `Run` must pass the
  [type proof](../../internal/checker/proof.go). A PID is a leaf of that proof.
- The type argument of `proc.Process` must pass the same proof, with one
  exception: `Prover.ArgumentReason` also accepts `proc.UncheckedDependency[T]` as the
  argument type, and named `proc.UncheckedDependency[T]` fields of an argument struct
  with exactly one marker.

`proc.UncheckedDependency[T]` is an [unchecked dependency](../processes.md#unchecked-dependencies).
The classifier treats the wrapper as protected storage and does not inspect `T`.
The type proof rejects it, so it cannot be in an Owner value, a frozen
collection, another marked struct, or an exported method signature. `checkStruct`
rejects a `proc.UncheckedDependency` field in an unmarked struct, which includes a
process. The source matcher rejects defined copies, pointers, arrays, native
containers, interface boxing, and package variables. Type arguments are checked
in the source walk so the explicit unchecked payload is skipped. `dependencyValue`
accepts `Value` only as a direct synchronous call on
`l.Args()` or on one field of it in a method of process `l`. The `T` value is
not proved; `proc.Trust` is the promise.
Before exempting an owned native handle, the source checker proves its protected
key and element types with the same Prover. Ownership alone does not admit an
argument-only type as a builder element.

A process is not an Owner holder; the holder pointer rules do not apply to it.
Methods with a process receiver are exempt from the rule against user-defined
methods with protected storage, because the four rules leave a direct static
call as the only use of such a method. The runtime creates a new value from the
same arguments after each restart delay and stores it in the same PID.
These rules do not check the origin or destination of mutable field handles
in ordinary method bodies; they do not prove memory isolation.

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
