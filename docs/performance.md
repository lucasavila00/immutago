# Performance

[Documentation](README.md) · [Collection API](../pkg/im/readme.md)

## Zero abstraction overhead

Equivalent native Go is the performance baseline. After inlining, the wrapper
should add no allocation, element copy, dynamic dispatch, runtime ownership
check, or extra heap escape. Freeze copies only a native handle.

| Operation | Storage work |
| --- | --- |
| `FreezeSlice`, `FreezeMap` | No allocation or element copy |
| `Len` | Constant time; no allocation |
| `At`, `Get` | Native indexing or lookup; no allocation |
| Direct range over `All()` | O(n); no container copy |
| `Clone` | O(n); independent mutable container storage |

Clone is shallow. Pointer elements still refer to the same immutable objects;
replacing a cloned slot does not change the frozen slot. There are no frozen
update methods. Edit a clone, then freeze it.

## Check costs

[Benchmarks](../pkg/im/benchmark_test.go) compare collection operations.
[Allocation tests](../pkg/im/iterator_test.go) separate native construction
from Freeze, reads, and direct iteration. Slice and map tests check storage
reuse and Clone independence. Run `make bench` for measurements.

For a cost change, compare equivalent native and `im` functions with the same
observable result. Check inlining, bounds checks, and heap escapes; inspect
machine code if results differ. Test local use and values returned by helpers.
Allocation counts alone do not prove zero overhead. Compare Clone with a native
container copy, not an in-place edit.
