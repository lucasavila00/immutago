# Performance

## Zero abstraction overhead

Equivalent native Go is the performance baseline. After inlining, the wrapper
should add no allocation, element copy, dynamic dispatch, runtime ownership
check, or extra heap escape for slice and map reads. `FreezeSlice` and `FreezeMap`
copy only a native handle. `FreezeString` explicitly copies bytes.

| Operation | Storage work |
| --- | --- |
| `FreezeSlice`, `FreezeMap` | No allocation or element copy |
| `FreezeString`, `FrozenString.Clone` | O(n) time and space; independent bytes; one allocation for a nonempty escaping copy, none for an empty copy |
| `Len` | Constant time; no allocation |
| `At`, `Get` | Native indexing or lookup; no allocation |
| Direct range over `All()` | O(n); no container copy |
| Slice or map `Clone` | O(n); independent mutable container storage |
| `Owner.Load`, `Owner.Store` | Atomic pointer operation; no object copy |
| `Ref.Current` | One atomic load; no allocation |

Slice and map Clone are shallow. Pointer elements still refer to the same immutable objects;
replacing a cloned slot does not change the frozen slot. There are no frozen
update methods. Edit a clone, then freeze it.
