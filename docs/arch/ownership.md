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
[process matcher](../../pkg/analyzer/process.go) applies five rules:

- A process value or pointer is valid only as the receiver variable in a method
  of that process or as a direct `PID.Proc()` result. The receiver can select
  fields other than the marker and call methods directly. A `Proc()` result can
  only receive a direct synchronous call of an exported method other than `Run`
  that is declared on the process.
- The process type name can appear only in its declaration, pointer receivers,
  and the type argument of `proc.Start`, `proc.StartReserved`, `proc.Reserve`, and `proc.PID`.
- Each parameter and result of an exported method other than `Run` must pass the
  [type proof](../../internal/checker/proof.go). A PID is a leaf of that proof.
- The type argument of `proc.Process` must pass the same proof.
- `Process.Supervisor` is valid only as a direct synchronous call on the
  receiver of `Run`, outside function literals. Its result must be the first
  argument of a direct synchronous `proc.Start`, `proc.StartReserved`, or
  `proc.StartSupervisor` call, directly or through one local variable from `:=`
  that has no other use. The supervisor of a run therefore does not escape the
  run, and only the runtime stops it.

A process is not an Owner holder; the holder pointer rules do not apply to it.
Methods with a process receiver are exempt from the rule against user-defined
methods with protected storage, because the five rules leave a direct static
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
