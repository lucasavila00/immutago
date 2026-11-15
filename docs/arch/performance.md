# Runtime costs and performance checks

Immutago removes repeated data copies by sharing proved immutable storage.
That choice moves work to construction, explicit Clone operations, root
publication, and analysis. A performance review must identify which of those
costs changed and measure the affected operation.

## Runtime cost model

| Operation | Work done by the current representation |
| --- | --- |
| FreezeSlice or FreezeMap | Copy a native handle into a private wrapper. No entry copy or runtime ownership check. Native builder construction has its own cost. |
| FreezeString | Keep the native immutable string. No byte copy or consumption of the input string. |
| Frozen collection read or iteration | Read native storage. No collection copy. |
| Clone | Allocate independent native collection slots and copy keys and values. Referenced immutable objects remain shared. |
| ProjectionOf | Copy its T into a projection. No backing-storage copy. T can be a root pointer or a larger value. |
| Generated field read | Read through a retained root pointer. Nested record and array readers retain the root and indices instead of copying parent records. |
| CopyRecord | Make an explicit private root copy; a nonzero-sized root needs one allocation. |
| Cell.Store | Publish a private root copy through an atomic pointer; a nonzero-sized T needs one allocation. Backing storage remains shared. |
| Cell.Load and LoadShared | Copy T by value. No backing-storage copy. |
| Cell.LoadPointer and generated Cell.Load | Load the root pointer without copying T. |
| Ref.Borrow | One atomic load from the shared reference slot. No allocation. |

These are representation costs, not a claim that every call has identical
machine code to native Go. Inlining, bounds checks, root size, escape analysis,
and call context can affect the result. A non-allocating large record copy can
still cost much more than a pointer load.

[Collection benchmarks](../../pkg/im/benchmark_test.go) compare native and im
operations. [Projection benchmarks](../../pkg/im/projection_benchmark_test.go)
measure local reads and field selection. [Generated read benchmarks](../../internal/projectiongen/testdata/readcost/README.md)
compare scalar and array reads from 8 KiB and 64 KiB records with native atomic
pointer reads. [Iterator tests](../../pkg/im/iterator_test.go) separate
construction allocations from Freeze and direct iteration. Slice and map tests
check storage identity and Clone independence, which timing alone cannot show.

For a runtime change, compare operations with the same observable result.
Compare Clone with a native container copy. Measure construction separately
from reads. Check both local use and helper returns, since escape analysis can
change at that boundary. If the result differs, inspect compiler inlining,
heap escapes, bounds checks, and, when needed, machine code.

## Analyzer cost model

The analyzer retains caches for one package pass. They trade pass memory for
repeated type and storage queries. They must not become shared mutable state
between concurrent passes.

| Structure | Work and retained data |
| --- | --- |
| Package selection | One seen set across local type roots. It follows used contracts instead of scanning the import graph's implementations. |
| Classification | One node per reached type and query mode, with reverse edges. Expand before propagating positive answers so cycles do not cache false negatives. |
| Type proof | One obligation per type and category. Reuse completed proofs; propagate failure through reverse edges. Keep one real cause per invalid node. |
| Failure rendering | Follow retained cause labels and cache requested messages. Empty pointer and array labels do not add message-chain work. |
| Storage query | Walk one selector/index/dereference/parenthesis chain for that query. Preserve the owner and builder context rather than restarting a recursive search at every step. |
| Cell queries | Cache completed slot results, including negative results for visited types. Reuse them through holder wrappers. |
| Builder flow | Dense owner IDs and input/output bitsets per event. Repeated reverse-postorder passes continue until consumed sets stop changing. |

Type graphs and rendered messages stay in memory until the pass ends. A bound
on one cache does not describe all compiler-type inspection or the complete
analyzer. In particular, flow cost depends on events, owners, graph edges, and
iterations. Do not infer full-pass complexity from a smaller diff or one timing.

`make bench-core` runs the `BenchmarkCore` cases in checker, freeze, and
analyzer packages. Parsing and Go type checking are outside the timed core
loops. The cases increase type graph size, invalid proof roots, selector depth,
slot graphs, or builder count so repeated traversal is visible. Relevant tests
also check the answer: [proof queries](../../internal/checker/prover_test.go),
[classification](../../internal/checker/classification_test.go),
[storage chains](../../pkg/analyzer/projection_test.go), and
[ordinary type graphs](../../pkg/analyzer/type_scaling_test.go).

## Dependency facts and driver memory

A failed package exports one original cause: package, portable source location,
and exact diagnostic message, plus a relevance bit. Importers forward that
cause, not the complete list of failures or import paths. This bounds the fact
payload independently of the number of failures behind it. It does not make
the analysis driver's import graph free.

Let P be the number of failed packages, E the number of direct invalid import
edges, and B the largest fact size, including path and diagnostic lengths.
Serialized fact payload is O(P × B). Reading facts across direct import edges
can require O(E × B) bytes. The CLI driver's inherited fact maps can retain
O(P²) pointers. These bounds exclude other driver state and compiler work.
Messages are not truncated, so B is not a fixed byte constant.

[Dependency graph tests](../../pkg/analyzer/dependencies_graph_test.go) check
cause selection, exact gob round trips, and fact size on deep, wide, and dense
graphs. A consumer-only test must still report the original cause when wrappers
hide the leaf from the driver's selected output.

## How to check a performance change

First keep a reproduction that produces the reported cost. Check its result
before measuring: a faster analyzer that skips required checks is a defect.
Choose the benchmark for the suspected structure, then compare the same cases
at baseline and changed revisions with the same Go version and runner setup.
Record repeated samples, allocation counts, and profiles when the cost needs
explanation. Keep the reproduction, revisions, environment, and before/after
measurements in the pull request.

Use these hosted workflows for full measurements:

- [Core performance](../../.github/workflows/core-performance.yml) runs repeated
  core benchmark samples and profiles large proof, builder, matcher, type, and
  slot cases. Artifacts contain the binary, CPU and allocation profiles,
  benchmark output, and environment. Run it on both revisions for a comparison.
- [Dependency performance](../../.github/workflows/dependency-performance.yml)
  accepts a baseline revision. It measures fact operations and complete dense
  modules through CLI and vet, with separate fresh caches for each case. It
  records total and largest fact sizes, elapsed time, peak memory, diagnostics,
  and profiles. Ordinary graphs must pass both drivers; checked graphs have
  known invalid-field and consumer-cause counts.

Profiles can include setup even when benchmark allocation counts exclude it.
Inspect that difference before attributing a cost to the timed core operation.
Hosted timing is evidence, not a fixed pass/fail threshold. Routine CI runs
correctness tests, not these benchmarks.
