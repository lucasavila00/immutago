# Architecture

Immutago combines compiler type proofs, source-form checks, and native builder
consumption checks. The analyzer adds no runtime state.

## Local review

The design reduces how much code a reviewer must trace. Put enforceable rules
at a type or API boundary; review implementation behavior inside that boundary.

| Boundary | What it removes from a caller review |
| --- | --- |
| Marked types and frozen collections | Searching other callers for writes to the same protected data. |
| Scoped pointers | Tracing checked callees to see whether they save the pointer. |
| Service Refs and result types | Inspecting callers for direct field access or mutable return values. |

Review service state changes, synchronization, and input contracts in the service
implementation. Ordinary inputs and dependency calls follow Go contracts. The
checker does not prove service logic or full isolation. Do not add whole-program
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
- [Storage](ownership.md): why we use type markers, pointer rules, and builder flow.
- [Validation](validation.md): trust boundary, test sources, and hosted checks.
- [Performance checks](performance.md): benchmarks, profiles, and checker costs.

The [language guide](../rules.md) links the complete allowed forms and examples.
The [review requirements](../reviewer-focus.md) explain how to trace each form
from that contract through the matcher and tests.
