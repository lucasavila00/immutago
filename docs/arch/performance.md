# Performance checks

## Check costs

| Evidence | What it measures |
| --- | --- |
| [Collection benchmarks](../../pkg/im/benchmark_test.go) | Native Go versus im operations. Run `make bench`. |
| [Owner benchmarks](../../pkg/im/owner_test.go) | Slot loads. |
| [Service benchmarks](../../pkg/supervise/supervise_test.go) | Ref.Current. |
| [Allocation tests](../../pkg/im/iterator_test.go) | Construction separately from Freeze, reads, and direct iteration. |
| [Frozen string tests and benchmarks](../../pkg/im/string_test.go) | Input and copy-out storage independence, empty substrings, and escaping copy allocations. |
| Slice and map tests | Storage reuse and Clone independence. |

For a cost change:

1. Compare native and im functions with the same observable result.
2. Check inlining, bounds checks, and heap escapes. Inspect machine code if costs differ.
3. Test local use and returns from helpers. Allocation counts alone do not prove zero overhead.
4. Compare Clone with a native container copy, not an in-place edit.

## Checker costs

`make bench-core` measures type proofs, builder flow, source matching, and the
package pass at increasing sizes. Parsing and Go type checking are outside the
timed loop. Stress tests compare results with independent reachability calculations.

| Cases | Coverage |
| --- | --- |
| [Storage projections](../../pkg/analyzer/projection_test.go) | Increasing selector/index/dereference/parenthesis depth. Each query visits each step at most once. |
| [Classification stress tests](../../internal/checker/classification_test.go) | Cycles, query order, aliases, marker identity, constraints, callback storage; independent storage and callback walks. |
| [Classification benchmarks](../../internal/checker/classification_benchmark_test.go) | Repeated queries at 16, 64, and 256 types. |
| [Ordinary type graphs](../../pkg/analyzer/type_scaling_test.go) | Full pass over connected unmarked structs and interfaces at 8, 16, 32, and 64 types. |
| [Slot queries](../../pkg/analyzer/slot_scaling_test.go) | Value-linked cycles at 512, 1024, and 2048 types in ordinary and checked packages. |
| [Marked graphs](../../pkg/analyzer/proof_scaling_test.go) | Full pass over marked types and frozen collections, with valid and many-invalid-root cases. |
| [Proof queries](../../internal/checker/prover_test.go) | Reused validity versus independent reachability; cause paths versus compiler type components. |

Cache costs:

- Classification expands each reachable type and mode once per cache; reverse
  edges propagate positive results through cycles.
- Completed negative slot queries cache every visited type. Wrappers reuse
  those results.
- Each proof obligation expands once; failure edges are processed a bounded
  number of times. Empty pointer/array labels add no rendering chain.
- Rendering follows retained message size; repeated queries reuse the message.
- Graphs and messages stay in memory until the pass ends.
- These bounds exclude other compiler-type inspection and the full analyzer pass.
  Structural and source checks run only in selected packages.

Run [Core performance](../../.github/workflows/core-performance.yml) manually:

- Compare the same cases and Go version before and after a change.
- Artifacts include repeated samples, CPU/allocation profiles, binary, and runner data.
- Profiles can include setup; benchmark allocation counts exclude it.
- Timing alone is not a pass/fail gate.

## Dependency fact costs

A failed package exports one cause (origin package, location, exact message) and
one relevance bit. Consumer reports name that cause and the origin to check;
direct package diagnostics remain complete.

| Quantity | Cost |
| --- | --- |
| P failed packages, largest fact B bytes | O(P × B) serialized payload. |
| E direct invalid import edges | O(E × B) bytes read by vet. |
| CLI driver's inherited fact maps | Can hold O(P²) pointers. |

B includes diagnostic and path lengths; messages are not truncated. These bounds
cover fact payloads, not all driver data or import-graph work.

Run [Dependency performance](../../.github/workflows/dependency-performance.yml)
manually with a baseline revision:

- Fresh vet facts on dense graphs; record total/largest fact size, time, peak memory, and profiles.
- Ordinary graphs without im imports must pass both drivers.
- Checked graphs import im and contain 50 invalid immutable fields per package;
  consumers report one cause.
- [Graph tests](../../pkg/analyzer/dependencies_graph_test.go) check cause selection,
  exact gob round trips, and constant fact size across deep, wide, and dense graphs.
