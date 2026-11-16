# Operation costs

| Operation | Work |
| --- | --- |
| FreezeString | Copy a native string handle; no byte copy, allocation, or input consumption. |
| FreezeSlice, FreezeMap | Copy a native handle; no entry copy or allocation. |
| Projection.Unsafe | Return T by value with its shared storage; no allocation. A pointer payload remains the same pointer. |
| Generated record projection.Unsafe | Return the selected pointer; no record copy, allocation, or Cell load. |
| ProjectionOf | Copy the Go value T; no copy of its backing storage. |
| Generated record field or array element projection | Read through the root pointer and indices; no copy of the containing record or array. |
| Value | Return a Copied value, opaque handle, pointer to a Frozen handle wrapper, or unnamed array of these values. Records and arrays have their ordinary Go value-copy cost. No allocation. |
| CopyRecord | Copy one record into a new root; one allocation for a record with nonzero size. Backing storage remains shared. |
| ProjectString, frozen Projection methods, generic projection adapters | Copy a native handle; no backing-storage copy or allocation. |
| FrozenString.Len/At, StringProjection.Len/At, SliceProjection.Len, MapProjection.Len/Get | Native length, indexing, or map lookup; Get wraps its returned value. |
| FrozenString.Equal/EqualString, StringProjection.Equal/Is | Native string equality; no allocation or byte copy. |
| FrozenString.Unsafe, StringProjection.Unsafe | Copy the native string handle; no allocation or byte copy. |
| SliceProjection.At | Copy one element into a Projection. |
| SliceProjection.AtPointer | Return a projection of one record's address without copying the record or allocating. |
| Direct range over All | O(n); projection iterators copy keys/elements as Go values. |
| FrozenString.Clone, StringProjection.Clone | O(n) byte copy; a nonempty escaping result uses one allocation. |
| Shared FrozenSlice/FrozenMap.Clone | O(n) shallow copy into independent native storage. |
| Local slice/map Clone | O(n); arguments must be Copied. Opaque handles are not Copied. |
| Cell.Store | Copy one T root and publish with an atomic store; one allocation for T with nonzero size. |
| Generated Cell.Load, raw Cell.LoadPointer | One atomic pointer load; no record copy or allocation. |
| Raw Cell.Load | One atomic load and a T value copy; no backing-storage copy. |
| Ref.Borrow | Return the stored pointer; no allocation. |
| Ref.IsZero | Compare the stored pointer with nil; no allocation. |

Copied record fields contain no strings or reference handles. A complete copy
includes all their data. The zero-sized im.Copied marker adds no runtime copy
operation or ownership state.

Large record and array values can cost more to copy than native pointer reads.
Do not describe these copies as free. The wrapper APIs add no automatic deep
copy, runtime ownership check, reflection, or string-origin tracking.

FreezeString keeps the native string's storage, including any original
allocation retained by a substring. FreezeSlice and FreezeMap keep the builder's
storage. Shared collection clones can still contain
pointers and nested shared collections. Local Clone requires the general
Copied constraint; otherwise copy each required field explicitly.

Projections keep their selected storage alive during use. Opaque service references
keep their separate shared-handle behavior.

## Frozen interface projections

A generated Frozen interface projection stores an interface header. Constructing
or copying the projection copies that header in O(1) time. A generated method
adds one direct interface call and no allocation of its own. The implementation's
method body and the original concrete-to-interface conversion keep their ordinary
Go costs. A Frozen record getter reads only the selected interface field; it
does not copy the parent record.

The [interface cost fixture](../../internal/projectiongen/testdata/interfacecost/README.md)
checks representation size and allocations and compares direct and projected
method calls.

## Market maker event path

The [market maker cost tests](../../examples/trading/cost_test.go) measure the
complete single-instrument event loop with a fixed-storage test adapter. They
check zero steady-state allocations and verify fresh price updates, order
cancellation and placement, and cumulative fills. The direct Go comparison uses
the same calculation, account locks, order counter, stream, and adapter work,
with a fixed service pointer and an atomic snapshot read in place of Ref and Cell reads.

Setup, periodic reports, network latency, and contention are outside that
comparison. Run the manual [Core performance workflow](../../.github/workflows/core-performance.yml)
with `scope: trading`; keep measured results in the relevant pull request.
