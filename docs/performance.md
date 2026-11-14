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
| `Owner.Load`, `Owner.Store` | Atomic pointer operation; no object copy |

Clone is shallow. Pointer elements still refer to the same immutable objects;
replacing a cloned slot does not change the frozen slot. There are no frozen
update methods. Edit a clone, then freeze it.

## Check costs

[Benchmarks](../pkg/im/benchmark_test.go) compare collection operations.
[Owner benchmarks](../pkg/im/owner_test.go) measure slot loads.
[Allocation tests](../pkg/im/iterator_test.go) separate native construction
from Freeze, reads, and direct iteration. Slice and map tests check storage
reuse and Clone independence. Run `make bench` for measurements.

For a cost change, compare equivalent native and `im` functions with the same
observable result. Check inlining, bounds checks, and heap escapes; inspect
machine code if results differ. Test local use and values returned by helpers.
Allocation counts alone do not prove zero overhead. Compare Clone with a native
container copy, not an in-place edit.

## Checker costs

`make bench-core` measures type proofs, builder flow, source matching, and the
full package pass at increasing input sizes. Parsing and Go type checking are
outside the timed loop. Stress tests compare recursive proofs and flow results
with independent reachability calculations.

[Core performance CI](../.github/workflows/core-performance.yml) saves repeated
benchmark samples, CPU and allocation profiles, the test binary, and runner
information as artifacts. Profile setup can appear in the profiles; benchmark
allocation counts exclude setup. Compare the same cases and Go version before
and after a change. Timing alone is not a pass/fail gate.

[Classification stress tests](../internal/checker/classification_test.go) compare
cached results with independent storage and callback walks and a reachability
oracle. They check cycles, query order, aliases, marker identity, constraints,
and callback storage. [Classification benchmarks](../internal/checker/classification_benchmark_test.go)
measure repeated queries at 16, 64, and 256 types. The
[ordinary type graph benchmark](../pkg/analyzer/type_scaling_test.go) measures the
full analyzer pass on connected structs and interfaces without markers at 8,
16, 32, and 64 types. These cases keep repeated graph traversal visible in CI.

The pass-local type classifier expands each reachable type and classification
mode once per cache. Reverse edges propagate positive results through cycles;
completed queries reuse the stored result. The graph stays in memory until its
pass ends. Ordinary packages still receive structural and source checks.
