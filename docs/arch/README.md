# Architecture

Immutago combines compiler type proofs, source-form checks, and native builder
consumption checks. The analyzer adds no runtime state.
These pages describe the implementation. The [checker reference](../rules.md)
defines supported Go forms; the [glossary](../glossary.md) defines shared terms.

## Local review

The goal is to reduce how much code a reviewer must trace.

| Concern | Where to check it |
| --- | --- |
| Immutable fields and collection elements | Type declarations and the im type proof. |
| Stored Local values and pointers | The im storage rules and their source checks. |
| Calls through service references | The Current call matcher and result type checks. |
| Service state changes, locks, and input use | The service code and ordinary Go API contracts. |

The checker does not prove full service isolation. Do not add whole-program
effect analysis to make that claim.

## Code map

| Source | Responsibility |
| --- | --- |
| [cmd/immutago](../../cmd/immutago/main.go) | Run Go's analysis driver. |
| [cmd/immutago-gen](../../cmd/immutago-gen/main.go) | Generate typed projections and cells for one package. |
| [internal/projectiongen](../../internal/projectiongen/generate.go) | Build canonical generated output from compiler-resolved types. |
| [pkg/analyzer](../../pkg/analyzer/analyzer.go) | Match source forms, report errors, and carry dependency failures. |
| [internal/checker](../../internal/checker/proof.go) | Prove each field, key, and element type. |
| [internal/freeze](../../internal/freeze/freeze.go) | Require one native builder handle and reject use after Freeze. |
| [internal/freeze/flow](../../internal/freeze/flow/graph.go) | Supply the builder solver's graph and set operations. |
| [pkg/im](../../pkg/im/marker.go) | Define `im.Frozen`, `im.Local`, and the trusted native storage wrappers. |
| [pkg/supervise](../../pkg/supervise/supervise.go) | Define `Service`, service references, and the trusted group runtime. |
| [tests](../../tests/cli_test.go) | Run consumer fixtures through the command and vet driver. |

## Design pages

- [Pipeline](pipeline.md): pass order, recursive type proofs, and dependency facts.
- [Storage](ownership.md): type markers, pointer rules, and builder flow.
- [Services](services.md): result and reference checks, service publication, and child lifetimes.
- [Validation](validation.md): trust boundary, test sources, and hosted checks.
- [Performance checks](performance.md): benchmarks, profiles, and checker costs.
