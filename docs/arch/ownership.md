# Storage and ownership

`FreezeSlice` and `FreezeMap` retain native handles. The analyzer requires one
mutable builder handle before the runtime shares that storage.

## From a native builder to frozen storage

```go
values := []int{10, 20}
frozen := im.FreezeSlice(values)
```

| Analysis stage | Data retained | Required result |
| --- | --- | --- |
| [Discover builders](../../internal/freeze/discover.go) | Compiler variable identity, declaration, function body, fresh initializer. | Track protected elements and all native Freeze inputs. |
| [Check uses](../../internal/freeze/uses.go) | Each use of the tracked variable. | No second handle, retained slot address, or unmatched escape. |
| [Check flow](../../internal/freeze/flow.go) | Ordered events and one bit per builder. | No use after Freeze on any incoming path. |

A scalar Clone that is never frozen follows ordinary Go rules. The
[collection reference](../reference/frozen.md) defines the permitted builder forms.

The flow solver assigns each builder a dense integer ID, then expands Go's
control-flow graph into events in evaluation order. Its consumed set records
builders whose storage may already have been frozen:

| Event | Example | Change to the consumed set |
| --- | --- | --- |
| Fresh declaration | `values := []int{10, 20}` | Clear the builder's bit, including on each loop execution. |
| Use | `len(values)` | Report if the incoming set contains the builder. |
| Freeze | `im.FreezeSlice(values)` | Set the bit after argument evaluation. |
| Branch join | One branch froze `values`. | Union incoming sets; a later use fails. |

The solver visits reachable events in reverse postorder until sets stop changing.
Delayed writes occur after their operands. An active range retains its handle,
so Freeze inside that range fails. Source matching receives the builder set and
still checks each operation.

## A field keeps its storage owner's rules

[One storage walk](../../pkg/analyzer/subset_shapes.go) follows selectors, indices,
dereferences, and parentheses. It carries the owner into write and receiver checks.

Assume `record` is Frozen and its `Totals` field is Copied:

| Operation | Storage being changed | Decision |
| --- | --- | --- |
| `state.Record = selected` | An ordinary Frozen pointer slot. | Permit pointer replacement; retain the target's rules. |
| `record.Totals.Count++` | A field inside `record`. | Reject the protected write. |
| `copy := record.Totals; copy.Count++` | An ordinary independent copy. | Use Go mutation rules. |

A method's implicit pointer receiver can change the same field. The checker
therefore examines the declared receiver before accepting an ordinary method call.
Frozen data fields need names, so Go cannot promote a mutating field method
to the Frozen record. Pointer handles and their ordinary containers use Go
transport rules; selections and dereferences still identify the protected target.
[Pointer and placement rules](../reference/rules.md) supply the complete source forms.

## Cell publication and retained projections

After Store replaces A with B:

```text
viewA ──> Rates A ──> frozen data A
Cell  ──> Rates B ──> frozen data B
viewB ───────┘
```

| Runtime operation | Stored or returned representation |
| --- | --- |
| [Cell.Store](../../pkg/im/cell.go) | A private copy of T behind `atomic.Pointer[T]`; referenced frozen storage stays shared. |
| Generated Cell.Load | A Local projection around `Projection[*T]`; no copy of T. |
| Raw Cell.Load | `Projection[T]` containing a copy of T; no copy of backing storage. |
| Generated projection.Unsafe | The selected *T; no record copy, allocation, or second Cell load. |
| Raw Projection.Unsafe | T by value; referenced storage stays shared. |
| A retained projection | Its pointer keeps the selected value reachable for Go's garbage collector. |

Cell source checks prevent copies of the atomic slot. The cached slot query follows
value fields and containers, and stops at pointers. Holder pointers use ordinary Go
transport. A holder value can select a field, form an address, or select a pointer
method without a copy. The checker also checks resolved method receivers to reject
implicit value copies. Projections need no registration or runtime ownership flag.

## Generated projections

For a Frozen Rates record with a frozen integer slice, the getter reads directly:

```go
func (v RatesProjection) Values() im.SliceProjection[int] {
    return v.immutagoProjection.Unsafe().Values.Projection()
}
```

| Selected data | Projection representation | Copy made |
| --- | --- | --- |
| Nested Frozen record | Pointer to the selected field. | No parent record copy. |
| Array | Retained record pointer and required indices. | No parent record or array copy. |
| Frozen record in a slice | `AtPointer` selects the element address. | No element copy. |
| Record returned by map lookup | `CopyRecord` creates a private addressable record. | Explicit record copy; map slots cannot supply an address. |

[The generator](../../internal/projectiongen/generate.go) loads source without stale
output, supplies provisional projection names, then resolves complete compiler types.
[Field generation](../../internal/projectiongen/readers.go) uses those types to write
getters and indexed array projections.

[The analyzer verifies the output](../../pkg/analyzer/generation.go) before source checks:

1. Generate expected bytes from the current types.
2. Compare the complete `zz_immutago.gen.go` after CRLF normalization.
3. Record its physical token file, unaffected by line directives.
4. Permit interior Frozen-record addresses only in that file.

The verified file still passes all other source checks.
See the [projection reference](../reference/projections.md) for generated declarations
and public projection forms.
