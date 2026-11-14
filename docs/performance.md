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
with independent reachability calculations. The [projection benchmark](../pkg/analyzer/projection_test.go)
measures one storage classification at increasing projection depths. Each query
visits each selector, index, dereference, or parenthesis at most once. This does
not bound the cost of the complete analyzer pass.

Run [Core performance](../.github/workflows/core-performance.yml) manually from
GitHub Actions when changing checker costs. It saves repeated
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
16, 32, and 64 types. These cases measure repeated graph traversal.

The [slot-query benchmark](../pkg/analyzer/slot_scaling_test.go) uses value-linked
cycles at 512, 1024, and 2048 types in ordinary and checked packages. Completed
negative slot queries cache every visited type. Recursive queries read that
cache, so new wrapper types do not expand the same negative component again.

The pass-local type classifier expands each reachable type and classification
mode once per cache. Reverse edges propagate positive results through cycles;
completed queries reuse the stored result. The graph stays in memory until its
pass ends. Ordinary packages still receive structural and source checks.

The [marked graph benchmarks](../pkg/analyzer/proof_scaling_test.go) measure the
complete pass on connected marked types and frozen collections.
`BenchmarkCoreMarked` has no errors; `BenchmarkCoreMarkedInvalid` measures many
invalid roots. [Proof-query tests](../internal/checker/prover_test.go) check
reused validity against independent reachability and check cause paths against
compiler type components.

Each concrete proof obligation is expanded once, and each failure edge is
processed a bounded number of times per cache. Empty pointer and array labels
do not add a chain of render work. Rendering is proportional to the retained
message text, and repeated queries reuse that message. The graph and cached
messages stay in memory until the pass ends. These bounds do not include all
compiler-type inspection or the complete analyzer pass.

## Dependency fact costs

Each failed package exports one representative original failure and a relevance
bit. The witness contains the origin package, source location, and exact message.
Downstream reports name that cause and the origin package to check; other errors
may exist. Reports from direct package analysis do not change.

For P failed packages, E direct invalid import edges, and a largest witness of
B bytes, serialized analyzer facts use O(P × B) bytes in total. Vet reads
O(E × B) bytes of these facts. B depends on the diagnostic and path lengths;
messages are not truncated. The CLI driver's inherited fact maps can still
hold O(P²) pointers. The payload bound does not bound all driver data or the
cost of the import graph.

Run [Dependency performance](../.github/workflows/dependency-performance.yml)
manually in GitHub Actions and select a baseline revision. It compares dense
graphs with fresh vet facts and records their total and largest file sizes,
timing, peak memory, and profiles. [Graph tests](../pkg/analyzer/dependencies_graph_test.go)
check the selected cause against all original errors, exact gob round trips,
and constant fact size across deep, wide, and dense graphs.
The workflow separately checks ordinary graphs without `im` imports and failure
graphs whose packages directly import `im`. Ordinary graphs must pass both
drivers. Checked graphs retain 101 local unsafe errors per generated package;
consumer reports contain one representative cause.
