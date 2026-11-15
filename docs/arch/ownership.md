# Storage, projections, and builder ownership

The runtime shares immutable backing storage instead of making a copy for each
reader. This requires three separate checks: prove the payload type, prevent
writes to a protected owner, and remove mutable aliases before Freeze. A
projection then limits what a reader can obtain from that storage.

## Payload types and storage owners

[The type prover](../../internal/checker/categories.go) has three recursive
categories. Each use requests the same category, regardless of its API context.

| Category | Meaning and consumers |
| --- | --- |
| Frozen | Safe for shared persistent storage. Used for Frozen fields, frozen collection elements, Cell payloads, and projection payloads. Includes booleans, numbers, strings, proved Frozen and Copied records, pointers to proved Frozen records, frozen wrappers and interfaces, service references, unchecked handles, and unnamed arrays of Frozen values. |
| Copied | Safe as an independent ordinary value. Used for Copied record fields and local projection Clone elements. Includes booleans, numbers, Copied records, unnamed arrays of Copied values, and opaque service references. Copied record fields additionally exclude service references. |
| ReadResult | Safe to return from a restricted reader. Used for Local fields and service data results. Includes Copied values, Local records, projections with Frozen payloads, unchecked handles, and unnamed arrays of read results. |

`im.Value` requires both Frozen and ReadResult. It can extract a scalar, Copied
record, unchecked handle, or unnamed array of such values. It cannot extract a
shared Frozen record or a Local value. A Frozen record is not Copied, even when
all its fields are numbers. Marker values themselves are annotations, not data
that satisfy these categories.

Type proof says what a value can contain. The owner of a field determines
whether that storage can change. Frozen and Local fields are fixed at
construction, including nested Copied fields and arrays. Copying a Copied field
into an ordinary variable removes that owner's write restriction:

```go
// totals is a Copied field in a Frozen record.
copy := record.totals
copy.Count++ // ordinary storage can change
// record.totals.Count++ would change protected storage.
```

[Storage classification](../../pkg/analyzer/subset_shapes.go) follows selector,
index, pointer, and parenthesis chains to find the owner. Write checks and
method receiver checks use that result. An implicit pointer receiver can change
an addressable field, so method matching checks the declared receiver type
before it takes an ordinary-call path. Copied values outside a protected owner
use normal Go mutation, pointers, interfaces, callbacks, and generic calls.

Protected pointers normally start at fresh storage, such as a record literal
or allocation. An address of a reassignable local would permit a later write
through that local, so it is rejected. Local pointers can remain local or pass
to direct typed calls, but cannot be stored or returned. Local values cannot
enter persistent storage through a record, array, collection, or Cell. These
are source restrictions; the runtime adds no loan or ownership state.

## Freeze transfers native storage

`FrozenSlice` and `FrozenMap` contain private native handles. Freeze puts the
existing handle in the wrapper; it does not copy the backing storage. The
analyzer must therefore establish one mutable builder handle and reject use
of it after transfer.

```go
values := make([]int, 0, 2)
values = append(values, 10, 20)
frozen := im.FreezeSlice(values)
// Later reads use frozen. A later use of values is rejected.
```

[Discovery](../../internal/freeze/discover.go) indexes variables by compiler
`*types.Var` identity, so shadowed names remain separate. It records each
builder's declaration, function body, and fresh initializer. Fresh sources are
native make, literals, bare nil, zero declarations, and the exact fixed Clone
methods. It selects fresh containers with protected elements and every native
Freeze input. A scalar Clone that is never frozen needs no builder analysis.

[Use checks](../../internal/freeze/uses.go) reject operations that can preserve
another handle: copies, subslices, slot addresses, escapes, and unmatched calls.
Reading an immutable pointer element does not alias the container slot. Taking
the slot's address does. The checks stop following storage when an operation
produces a separate value.

[Flow analysis](../../internal/freeze/flow.go) then checks consumption in each
participating function:

1. Build Go's control-flow graph and expand blocks into ordered events.
2. Give each builder a dense integer ID. Events carry that ID plus consume and
   reset flags.
3. Track a bitset of possibly consumed builders at each event. Merge incoming
   paths by union. Freeze sets its input bit after argument evaluation. A fresh
   declaration clears its bit, including each execution in a loop.
4. Report a use when its input state contains that builder's bit.

The solver visits reachable events in reverse postorder until states stop
changing. A Freeze on either branch makes a later joined use invalid. Delayed
writes occur after operand evaluation, so a Freeze in a write operand cannot
hide use of the old storage. Freeze during an active range over the same builder
is rejected because the range retains its handle.

Builder analysis returns the participating variable set to source matching.
That set permits native builder storage; it does not exempt its operations
from source checks. Exclusive use and consumption are both required. There is
no runtime fallback if analysis is omitted.

## A projection retains a read root

`Projection[T]` contains T by value. For a record reader, T is a pointer to the
Frozen root. A generated projection is an im.Local record around that pointer
projection. Its methods select fields directly from the retained root. Nested
record and array projections keep the root and necessary indices, so a scalar
read does not copy a large parent record.

[Generated field methods](../../internal/projectiongen/readers.go) select a
result representation by type. Copied values, service references, and unchecked
handles return directly. Frozen records and arrays return further projections.
Native strings use ProjectString; frozen string, slice, and map fields use their
Projection methods. Generated field getters do not expose a mutable slice, map,
or raw record pointer. Declared Frozen interface methods keep their ordinary result
rules and can return a Frozen pointer.

Collection projections can still copy an element: SliceProjection.At and
MapProjection.Get copy the element value into a Projection. SliceProjection.AtPointer
selects a Frozen record element by address without copying it.

A record returned by value from a map lookup has no addressable map slot.
`CopyRecord` makes an explicit private root copy and returns its pointer
projection. A retained nonzero-sized root needs heap storage; the compiler can
eliminate an allocation for local use. Clone makes independent native collection
slots under its type constraints; it does not clone referenced objects. Shared referenced objects
remain immutable. `StringProjection.Unsafe` and `FrozenString.Unsafe` explicitly
return a shared native string. The caller must obey string immutability; the
analyzer does not track string origins or lifetimes.

## Why generated code has two special permissions

Application code cannot call `Projection.Read`, even for a scalar. The exact
current generated file can call it and form interior Frozen-record addresses.
These operations let typed getters read fields without exposing a public raw
payload getter.

[Generation verification](../../pkg/analyzer/generation.go) requires named
package-level Frozen records and interfaces in non-test source, plus exactly
one canonical go:generate directive. It regenerates expected bytes from
compiler types and compares the complete file after CRLF normalization. It
then records the physical token file. A generated header, filename alone, or
line directive cannot grant permission. A stale or changed file gets neither
permission. The verified file still passes all normal type, write, capture,
generic, receiver, and transport checks.

The generator first loads source without stale generated declarations. It
supplies provisional projection names, reloads the package with those names,
then proves complete types and writes final methods. This lets interface method
signatures refer to generated Local projections, including recursive results.
A final compiler load checks changed provisional output.

Frozen interface projections retain the proved interface and forward declared
method calls. Each concrete implementation is proved at conversion, with exact
declared receivers and no promoted user methods. The generated projection has
no raw getter. A declared method can still return its receiver or other values
under ordinary method rules; there is no method-effect proof.

## Cell replaces the root

[Cell[T]](../../pkg/im/cell.go) contains an atomic pointer to T. Store takes a
private root copy and publishes its address. It copies T, not immutable backing
storage. LoadShared copies the current T for persistent use. Raw Load copies T
into a local Projection[T]. LoadPointer returns a pointer projection without
copying T; generated record Cell.Load uses this path.

A reader can retain an older projection after Store. Its root remains reachable
through the projection and Go keeps it alive. No explicit reader registration
or runtime ownership tracking is needed. Possession of the Cell also permits
LoadShared; projections do not return the Cell to a reader.

[Slot checks](../../pkg/analyzer/owner_slot.go) require named Cell slots and
restricted direct access, including holder records. Value-copy checks prevent
copies of the atomic slot. Holder-pointer source forms are also restricted by
the current matcher. Holder methods use exact pointer receivers, with the
allowed Go implicit address of an addressable holder variable or direct field. The
payload still uses the same Frozen proof as every other persistent value.

`UncheckedValue` is a deliberate exception to payload inspection. Its declaring
package promises immutability; Trust and Unwrap are restricted to that package.
The opaque handle is Frozen and a ReadResult, but not Copied. It can pass through
services and generated getters without exposing a local projection of its
payload. This promise does not add another proof category.

The [type reference](../reference/types.md) and [source reference](../reference/rules.md)
give the complete allowed forms. [Freeze tests](../../internal/freeze/freeze_test.go)
check aliases and flow; [generated read tests](../../internal/projectiongen/testdata/readcost/README.md)
check that field reads preserve the intended representation.
