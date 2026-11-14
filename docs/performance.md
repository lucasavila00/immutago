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

### Repeated type-query evidence

The [baseline run](https://github.com/lucasavila00/immutago/actions/runs/43543423962)
used commit `4653421`; the
[cached run](https://github.com/lucasavila00/immutago/actions/runs/43543955011)
used `8a8da5e`. Both used Go 1.27.1 on Linux/amd64. The table gives median
allocation counts from three samples of `BenchmarkCoreOrdinary`; MB and GB are
decimal units. Compiler setup is outside the measured pass.

| Types | Before, allocated bytes/pass | After, allocated bytes/pass |
| --- | ---: | ---: |
| 8 | 172.36 MB | 2.42 MB |
| 16 | 1.30 GB | 4.85 MB |
| 32 | 9.82 GB | 9.72 MB |
| 64 | 77.62 GB | 19.43 MB |

At 64 types, allocation count fell from 37,440,151 to 15,776 per pass. The
baseline allocation profile attributes 99.2% of allocated bytes to repeated
`contains` walks. The cached profile no longer has that cost; parent indexing
and constant-expression classification are its largest allocation sources.
Profiles can include setup; use the benchmark samples for per-pass counts.

The runners had different CPU models: AMD EPYC 9V74 before and AMD EPYC 7763
after. The cached samples had median times of 7.37, 15.38, 31.23, and 72.76 ms
at increasing sizes. These timings describe that run, not a fixed speedup ratio.
The cache stores graph nodes and edges until its analysis pass ends. It does not
skip ordinary packages or add state to immutable values.
