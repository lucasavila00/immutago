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

The checker knows that `limit` cannot save or return `config` because Config is
Scoped. Another function that receives `*Config` gets the same rule. Renaming
the parameter does not change it.

The marker selects the pointer rule:

- `im.Shared`: callers can store or return pointers.
- `im.Scoped`: callers can read through pointers, but cannot store them in fields
  or globals, capture them, or return them.

For either marker, the checker requires:

- Immutable fields, including nested fields and array elements.
- Immutable targets for stored pointers, so another pointer cannot change them.
- Allowed key and element types in frozen collections.
- No writes to those fields through another pointer or a receiver method.

This choice gives us one rule to declare, read, and check. The checker uses Go
type identity instead of a separate pointer policy for each variable and
parameter. It still checks each operation; the marker is not a trusted promise.

The cost is less flexibility: one type cannot have both Shared and Scoped
pointer rules. For example, a Shared type cannot also be used in `Owner[T]`.
Code that needs both roles must use separate types or pass value copies.

Parameter rules could let one type serve both roles. They would require more
annotations and checks for pointer copies and calls. Both designs enforce the
same immutable-field rule. We keep type markers unless a concrete use case shows a large
correctness or simplicity gain from that extra work.

See [Shared](../shared.md), [Scoped](../scoped.md), and [Owner](../owner.md) for
the supported forms.

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

## Services and groups

The service boundary has two parts: the analyzer checks source forms, and the
trusted [runtime](../../pkg/supervise/supervise.go) manages service values and
groups. These checks do not prove that ordinary method bodies isolate mutable
memory. A service can still reach a shared map through ordinary Go code.

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
   defer calls. The stored-type proof checks each parameter and result of the
   declared exported method.
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

A reference contains a pointer to an atomic pointer, so copied references select
the same current value. `StartReserved` creates a fresh service, binds its
arguments, and calls `Init`. Successful initial setup publishes with
compare-and-swap. A losing concurrent start cleans up its staged children before
it panics. Later successful restart attempts replace the current pointer with
an atomic store; failed setup leaves the previous value available to callers.
`Current` needs one atomic load and no allocation.

A run owns a distinct child group. Children created during `Init` are registered
but wait until the run begins. Failed setup stops those children. During shutdown,
the child group stops before the runtime cancels the parent's Run context. On a
normal return or panic, child cleanup finishes before the run ends or waits to
restart. A completed run keeps a stopped child group, so later method calls can
initialize children but cannot run them. The checker does not prove whether a
saved child reference belongs to the current parent run.

The source matcher accepts the fixed Start result forms when arguments contain
protected storage, including two local result bindings and a matching named
function return. Calls with only ordinary arguments retain ordinary Go binding
rules. The `Init` error result does not admit `error` into other exported service
method signatures. [CLI fixtures](../../tests/testdata/accepted/serviceinit/main.go)
and [runtime tests](../../pkg/supervise/init_test.go) check these separate rules.

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
