# Operation costs

| Operation | Work |
| --- | --- |
| FreezeString | Copy a native string handle; no byte copy, allocation, or input consumption. |
| FreezeSlice, FreezeMap | Copy a native handle; no entry copy or allocation. |
| ProjectionOf | Copy the Go value T; no copy of its backing storage. |
| Generated record field or array element projection | Read through the root pointer and indices; no copy of the containing record or array. |
| Copy | Return a Copied value; records and arrays have their ordinary Go value-copy cost. No allocation. Frozen records are not Copied. |
| CopyRecord | Copy one record into a new root; one allocation for a record with nonzero size. Backing storage remains shared. |
| ProjectString, frozen Projection methods, generic projection adapters | Copy a native handle; no backing-storage copy or allocation. |
| FrozenString.Len/At, StringProjection.Len/At, SliceProjection.Len, MapProjection.Len/Get | Native length, indexing, or map lookup; Get wraps its returned value. |
| FrozenString.Equal/EqualString, StringProjection.Equal/EqualString | Native string equality; no allocation or byte copy. |
| FrozenString.Unsafe, StringProjection.Unsafe | Copy the native string handle; no allocation or byte copy. |
| SliceProjection.At | Copy one element into a Projection. |
| SliceProjection.AtPointer | Return a projection of one record's address without copying the record or allocating. |
| Direct range over All | O(n); projection iterators copy keys/elements as Go values. |
| FrozenString.Clone, StringProjection.Clone | O(n) byte copy; a nonempty escaping result uses one allocation. |
| Shared FrozenSlice/FrozenMap.Clone | O(n) shallow copy into independent native storage. |
| Local slice/map Clone | O(n); arguments must be Copied. Service references remain opaque shared handles. |
| Cell.Store | Copy one T root and publish with an atomic store; one allocation for T with nonzero size. |
| Generated Cell.Load, raw Cell.LoadPointer | One atomic pointer load; no record copy or allocation. |
| Raw Cell.Load, Cell.LoadShared | One atomic load and a T value copy; no backing-storage copy. |
| Ref.Borrow | One atomic load; no allocation. |

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
