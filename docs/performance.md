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

The pass-local type classifier expands each reachable type and classification
mode once per cache. Reverse edges propagate positive results through cycles;
completed queries reuse the stored result. The graph stays in memory until its
pass ends. Ordinary packages still receive structural and source checks.

## Dependency fact costs

Dependency facts share local failure records and retain direct import edges.
For a graph with P packages, E invalid import edges, and F bytes of local error
data, the CLI retains O(P + E + F) record data plus the per-fact tables of reachable
package pointers. Those tables, and the driver's package-fact maps, can still
contain O(P²) pointers. Facts contain no per-error copies of full import chains.

Each serialized fact contains one copy of each reachable package record and
its local errors and edges. Vet can still repeat that data in different fact
files; exact original diagnostics require that data across silent wrappers.
Reports build routes only for relevant consumers. Report storage includes the
text that must be shown to the user; no errors are capped or discarded.

Run [Dependency performance](../.github/workflows/dependency-performance.yml)
manually from GitHub Actions when changing dependency fact costs. It measures merging, gob encoding, and reporting separately. It also compares the
base and changed commands on dense 16- and 64-package graphs, then checks 144,
256, and 400 packages with the changed command. Each package imports every
package in the preceding layer and has the issue's 50 unsafe conversions.
With the repository's Go version, these produce 101 local errors per package:
one import error and two source positions per conversion. Both drivers must
preserve the exact count. A separate relevant consumer must receive all 40,400
original errors through a silent wrapper.

The workflow saves timing, peak-memory, allocation, CPU-profile, and runner
data. The baseline 400-package run is omitted because the reported failure is
an out-of-memory exit. The graph tests also bound encoded size by one copy of
each record and compare reports with the previous propagation algorithm.

[Proof-query tests](../internal/checker/prover_test.go) compare reused validity
with independent reachability and exact diagnostics with the original fresh
proof. They include cycles, query order, aliases, repeated edges, terminal
failures, and new owners of failed children.
The [marked graph benchmarks](../pkg/analyzer/proof_scaling_test.go) measure the
complete analyzer on connected marked types, frozen slices and maps, and ten
construction/read sequences per function. `BenchmarkCoreMarked` has no errors;
`BenchmarkCoreMarkedInvalid` keeps the cost of many invalid roots visible.

The prover retains each reachable obligation and edge once per pass. Validity
expansion and failure propagation cost O(V + E) across its queries. Exact
messages for R distinct invalid roots can still require O(R × (V + E)) replay
work, plus the diagnostic bytes. Completed messages are cached by root and
retain O(total message bytes) until the pass ends. This is not a universal
linear bound for rejected programs. Parsing and type checking remain outside
the benchmark's timed pass.
