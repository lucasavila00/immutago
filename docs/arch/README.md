# Architecture

[Documentation](../README.md) · [Language contract](../rules.md)

Immutago combines compiler type proofs, source-form checks, and native builder
consumption checks. The analyzer adds no runtime state.

## Code map

| Source | Responsibility |
| --- | --- |
| [cmd/immutago](../../cmd/immutago/main.go) | Run Go's analysis driver. |
| [pkg/analyzer](../../pkg/analyzer/analyzer.go) | Match source forms, report errors, and carry dependency failures. |
| [internal/checker](../../internal/checker/proof.go) | Prove each field, key, and element type. |
| [internal/freeze](../../internal/freeze/freeze.go) | Require one native builder handle and reject use after Freeze. |
| [internal/freeze/flow](../../internal/freeze/flow/graph.go) | Supply the builder solver's graph and set operations. |
| [pkg/im](../../pkg/im/marker.go) | Define both markers and the trusted native storage wrappers. |
| [tests](../../tests/cli_test.go) | Run consumer fixtures through the command and vet driver. |

## Design pages

- [Pipeline](pipeline.md): pass order, recursive type proofs, and dependency facts.
- [Storage](ownership.md): immutable writes, pointer escape rules, and builder flow.
- [Validation](validation.md): trust boundary, test sources, and hosted checks.

The [language contract](../rules.md) owns the complete allowed forms and examples.
The [review requirements](../reviewer-focus.md) explain how to trace each form
from that contract through the matcher and tests.
