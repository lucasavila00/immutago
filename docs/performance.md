# Operation costs

## Zero abstraction overhead

Equivalent native Go is the performance baseline. After inlining, a frozen
wrapper should add no allocation, entry copy, dynamic dispatch, runtime ownership
check, or extra heap escape to reads. Freeze and `.ByValue()` copy only a native
handle. They do not copy entries or string bytes.

| Operation | Storage work |
| --- | --- |
| `FreezeSlice`, `FreezeMap`, `FreezeByValueSlice`, `FreezeByValueMap` | No allocation or entry copy; consumes the builder. |
| `FreezeString`, `FreezeByValueString` | No allocation or byte copy; input remains usable. |
| Shared frozen `.ByValue()` | No allocation or entry/byte copy; preserves type arguments. |
| `Len` | Constant time; no allocation. |
| `At`, `Get` | Native indexing or lookup; no allocation. |
| Direct range over `All()` | O(n); no container copy. |
| Shared slice or map `Clone` | O(n); independent mutable container storage. |
| Local slice `Clone` | O(n); independent mutable container storage; nested views are rejected. |
| Local map `Clone` | O(n + b); independent mutable map and text key bytes, where b is the total bytes in `FrozenString` keys; nested views are rejected. |
| Either string `Clone` | O(n) time and space; independent bytes. A nonempty escaping copy uses one allocation; an empty copy needs none. |
| `Owner.Load`, `Owner.Store` | Atomic pointer operation; no object copy. |
| `Ref.Current` | One atomic load; no allocation. |

Shared collection Clone is shallow. Pointers and nested shared frozen entries
still refer to the same immutable data. Local collection reads reject those
entry types. Service references keep their explicit shared handle behavior in
either family. No Clone copies a service.

A local view retains its source storage while it is in use. A shared frozen
value can retain it for as long as callers keep that value. String Freeze does
not detach a substring. Slice Freeze does not reduce the builder's capacity.
Use Clone when independent native storage is required.

Converting each element of `FrozenSlice[FrozenString]` to a local string view
requires an explicit O(n) builder of view headers. The strings are not copied.
See the [text-view example](local-views.md#example-a-list-of-text-views).

The [performance checks](arch/performance.md) describe benchmarks, allocation
tests, and comparisons with native Go.
