# Type rules, projections, and builder storage

## Type rules

The [proof graph](../../internal/checker/proof.go) keys each obligation by resolved
Go type and proof category. The same cached obligations serve all consumers.

| Proof | Consumers |
| --- | --- |
| FrozenReason | Frozen fields, shared frozen arguments, Cell[T], Projection[T], ProjectionOf. |
| CopiedReason | Copied record fields and local collection Clone arguments. |
| ReadResultReason | Local fields, service data results, and Value results (also requiring Frozen). |
| General immutable proof | Marked records and fixed library types, including Local values. |

Frozen, Local, and Copied markers set one field rule each. API names and service
context do not change those rules. Ordinary functions can return shared
Frozen values; APIs that require read results accept Copied values, Local values, unchecked handles, and unnamed arrays of read results.
Copied contains booleans, numbers, Copied records, unnamed arrays of Copied
values, and opaque service references. Copied records reject string and reference
fields, including service references. The cached handle query applies this field
rule before following Copied field obligations. Frozen records are not Copied.
Local fields and service data results both use this boundary.

## Source matching

| Rule | Implementation |
| --- | --- |
| Frozen and Local fields are immutable from construction | [Write matcher](../../pkg/analyzer/subset_writes.go). |
| Ordinary Copied storage uses normal Go operations | One storage classification excludes Copied records. The owner walk still finds Frozen and Local owners around Copied fields and elements. |
| Protected pointers originate from fresh storage | [Source matcher](../../pkg/analyzer/subset_operations.go); reject addresses of reassignable locals. |
| Local storage and captures cannot escape | Declaration, assignment, container, callback, and closure checks use the recursive Local classification. |
| Application code cannot call Projection.Read | [Generation checks](../../pkg/analyzer/generation.go) identify the exact current generated token file before source checks. Handwritten Read calls are rejected. |
| Reads preserve access types | Generated getters return Copied values, unchecked handles, or local projections. Value requires both Frozen and ReadResult. |

## Generated projections

The [generator](../../internal/projectiongen/generate.go) reads named package-scope
Frozen declarations with compiler type information. Frozen records get typed
projections, projection constructors, and Cell holders. Local and Copied records need no
generation.

[Generated field methods](../../internal/projectiongen/readers.go) use a projection of the
root pointer and read fields directly. Frozen records and arrays use projections with
that root and the required indices. Copied scalars/records, opaque service references, and unchecked handles
return directly. Native strings use ProjectString; frozen string, slice, and map
fields use their Projection methods.

The [generation check](../../pkg/analyzer/generation.go) requires the canonical
directive and compares normalized output bytes with the generator's current
result. Before source checks, it identifies the matching physical token file.
Only that file can call im.Projection.Read or form interior Frozen-record
addresses. A generated header or line directive gives no such permission.
All ordinary type, write, capture, generic, receiver, and transport checks remain.
Generated declarations are excluded from input types, not from source checks.
Function-local, anonymous, and test-only Frozen declarations are rejected.

## Cell capability

[Cell](../../pkg/im/cell.go) stores a private atomic pointer. Store creates a new
T root. Raw Load returns Projection[T]; LoadPointer returns Projection[*T]; LoadShared returns T.
The generated Cell.Load uses LoadPointer without a root copy. All reads use the
same Frozen payload proof. Only code with the Cell has LoadShared access.
A projection cannot recover the Cell or expose a raw Frozen-record pointer.
An unchecked handle can be read with Value; its declaring package can then use Unwrap.

The [slot matcher](../../pkg/analyzer/owner_slot.go) restricts Cells to named
slots and direct calls. Named holder fields can contain other holders. Holder
methods use direct non-generic calls and exact pointer receivers, or Go's
implicit address of an addressable holder variable or direct field. Holder
checks prevent copying the atomic pointer. Local
values cannot be stored in a Cell, even through a Frozen record or type alias.

## Trusted values

UncheckedValue is a Frozen handle under its declaring package's immutability
promise. It is also a read result, so Local fields, services, generated getters,
and Value can return the handle. It is not Copied; projection Clone keeps its
Copied requirement. Trust/Unwrap keep their declaring-package check. The handle
classification stays opaque, so ordinary handle transport keeps Go rules.

## Native builders

| Stage | Work |
| --- | --- |
| [Discovery](../../internal/freeze/discover.go) | Index local declarations by compiler variable identity. Track containers with protected contents and native Freeze inputs. Native Copied containers need ownership checks only when later frozen. |
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
- FrozenString holds a private native string. FreezeString does not copy bytes or consume the immutable input.
- Reads and [iteration](../../pkg/im/iterator.go) use the native storage directly.
- Clone copies slots, not referenced objects. Those objects remain safe to share because they are immutable.
