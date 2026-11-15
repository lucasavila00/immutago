# Architecture

Immutago combines compiler type proofs, source-form checks, and native builder
consumption checks. The analyzer adds no runtime state.

## Local review

The goal is to reduce how much code a reviewer must trace.

| Concern | Where to check it |
| --- | --- |
| Immutable fields and collection elements | Type declarations and the im type proof. |
| Saved Scoped pointers | The im pointer rules and their source checks. |
| Access through service handles | [Direct Current calls, copies, and storage](ownership.md#service-handle-boundary). |
| Service state changes, locks, and input use | The service code and ordinary Go API contracts. |

The checker does not prove full service isolation. Do not add whole-program
effect analysis to make that claim.

## Code map

| Source | Responsibility |
| --- | --- |
| [cmd/immutago](../../cmd/immutago/main.go) | Run Go's analysis driver. |
| [pkg/analyzer](../../pkg/analyzer/analyzer.go) | Match source forms, report errors, and carry dependency failures. |
| [internal/checker](../../internal/checker/proof.go) | Prove each field, key, and element type. |
| [internal/freeze](../../internal/freeze/freeze.go) | Require one native builder handle and reject use after Freeze. |
| [internal/freeze/flow](../../internal/freeze/flow/graph.go) | Supply the builder solver's graph and set operations. |
| [pkg/im](../../pkg/im/marker.go) | Define `im.Shared`, `im.Scoped`, and the trusted native storage wrappers. |
| [pkg/supervise](../../pkg/supervise/supervise.go) | Define the service marker, Refs, and the trusted group runtime. |
| [tests](../../tests/cli_test.go) | Run consumer fixtures through the command and vet driver. |

## Design pages

- [Pipeline](pipeline.md): pass order, recursive type proofs, and dependency facts.
- [Storage](ownership.md): type markers, pointer rules, and builder flow.
- [Validation](validation.md): trust boundary, test sources, and hosted checks.
- [Performance checks](performance.md): benchmarks, profiles, and checker costs.

The [language guide](../rules.md) links the complete allowed forms and examples.
The [review requirements](../reviewer-focus.md) explain how to trace each form
from that contract through the matcher and tests.
