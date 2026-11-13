# Architecture

[Documentation](../README.md) · [Language rules](../rules.md) · [Development](../development.md)

Immutago is a Go analysis pass plus a small collection library. It uses compiler
objects and source control flow to check types, immutable objects, and
native builders. It adds no runtime ownership state.

## Code map

| Component | Responsibility |
| --- | --- |
| [`cmd/immutago`](../../cmd/immutago/main.go) | Run the standard Go analysis driver |
| [`pkg/analyzer`](../../pkg/analyzer/analyzer.go) | Check source forms, produce diagnostics, and carry dependency failures |
| [`internal/checker`](../../internal/checker/proof.go) | Prove field and collection argument types recursively |
| [`internal/freeze/flow`](../../internal/freeze/flow/graph.go) | Store builder control-flow graphs |
| [`internal/freeze`](../../internal/freeze/freeze.go) | Prove exclusive native builders and reject uses after Freeze |
| [`pkg/im`](../../pkg/im/marker.go) | Provide the marker and trusted frozen storage wrappers |
| [`tests`](../../tests/cli_test.go) | Exercise the shipped CLI and vet driver with consumer fixtures |

## Read by question

- **How does source reach a diagnostic?** Read the [analysis pipeline](pipeline.md).
- **How do objects stay immutable and builders become consumed?** Read [ownership analysis](ownership.md).
- **What does the checker trust, and what verifies it?** Read [validation](validation.md).

## Why this design

Constructors can return values or shared pointers. Objects cannot change after
construction. A pointer starts from a fresh allocation, so rebinding a variable
cannot change an object through a retained address. References can be copied,
stored, and returned. Go manages their lifetime; the analyzer needs no loan
liveness or pointer-origin propagation.

Direct named functions can receive and return shared references. Calls must
match the complete source forms in the guide. The checker does not infer
arbitrary callback effects or permit writable pointer targets.

Freeze transfers a proved exclusive native handle to an `im` wrapper. Clone
copies that storage for further edits. There are no persistent update trees
or mutable borrows. The checker recognizes the exact library types; it does
not prove arbitrary storage libraries with native reference fields.

Each accepted operation must pass both the source-form match and its ownership
checks. Dependency failures must reach the consumer. See the
[review requirements](../reviewer-focus.md) before changing these checks.
