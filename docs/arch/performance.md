# Performance

The [operation-cost reference](../reference/performance.md) covers public im calls.
The structures below determine analysis memory and repeated work.

## State retained for one package pass

| Structure | Retained data | Cost to inspect |
| --- | --- | --- |
| Package selection | One visited-type set across local roots. | Reachable used contracts. |
| Type proof | Nodes by type and category; reverse edges; one cause per failed node. | Graph expansion, failure propagation, requested message rendering. |
| Classification | Nodes by type and query mode; reverse edges. | Graph expansion and propagation of positive answers. |
| Storage queries | Owner and builder context from one selector/index chain. | Chain length for each query. |
| [Cell queries](../../pkg/analyzer/owner_slot.go) | Completed answers, including negative answers for visited types. | New types reached before a cached answer. |
| [Builder flow](../../internal/freeze/flow.go) | Dense owner IDs; input and output bitsets for every event. | (Events + edges) × bitset words × solver iterations. |

Caches end with the pass. They are not shared mutable state between concurrent
passes. Builder bitset width is `ceil(builderCount / 64)`; convergence depends on
the control-flow graph.

## Dependency facts and driver memory

Each failed package exports one original cause, including its full message.
Let P be failed packages, E direct invalid import edges, and B the largest fact size.

| Storage or transfer | Bound | Included data |
| --- | --- | --- |
| Serialized facts | O(P × B) | One cause per failed package. |
| Reading direct-import facts | O(E × B) | Fact payload across each invalid import edge. |
| CLI driver's inherited fact maps | O(P²) pointers | References to facts from indirect dependencies. |

B includes path and message lengths; it is not a fixed byte constant. These bounds
exclude other driver state and compiler work. [Graph tests](../../pkg/analyzer/dependencies_graph_test.go)
check serialization and fact sizes on deep, wide, and dense imports.

## Choose evidence for the changed cost

| Changed work | Measurement | Comparison |
| --- | --- | --- |
| Collection operations | [im benchmarks](../../pkg/im/benchmark_test.go) and representation tests. | Equivalent native operation; separate construction from reads. |
| Generated field reads | [Read-cost cases](../../internal/projectiongen/testdata/readcost/README.md). | Native pointer reads from the same 8 KiB and 64 KiB records. |
| Type, matcher, or builder work | [Core workflow](../../.github/workflows/core-performance.yml). | Repeated benchmark samples and CPU/allocation profiles. Parsing and type checking are outside timed core loops. |
| Dependency propagation | [Dependency workflow](../../.github/workflows/dependency-performance.yml). | Fact bytes, complete CLI/vet runs, peak memory, and correct diagnostics with fresh caches. |

1. Keep a reproduction and verify its result.
2. Compare baseline and changed revisions with the same Go version and runner setup.
3. Record repeated time/allocation samples; use profiles to explain the changed cost.
4. Put the reproduction, environment, revisions, and measurements in the pull request.

Profiles can include setup excluded from benchmark timing. Routine CI checks
correctness; it does not set benchmark thresholds.
