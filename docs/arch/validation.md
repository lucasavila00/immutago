# Validation boundary

Tests check both the analyzer's decisions and the runtime contracts that make
those decisions useful.

## Trusted parts

| Part | Assumption or check |
| --- | --- |
| Go compiler | Supplies valid ASTs, resolved types, object identities, and method selections. |
| Fixed im and supervise packages | Their reviewed implementations obey API contracts; their source is excluded from analysis. |
| Generated readers | Verify exact bytes for interior Frozen-record addresses; retain all other source checks. |
| Ordinary Go APIs | Obey their contracts, including string immutability and callback behavior. A Frozen pointer must not enter an API that changes its target; reflection and unsafe mutation effects are not proved. |
| `UncheckedValue[T]` | The declaring package promises that the opaque payload never changes. |

The [package boundary](../reference/rules.md#package-boundary) defines selected source.
A normal Go build adds no Freeze enforcement. Each relevant build configuration
must run the analyzer.

## Evidence for each mechanism

| Mechanism | Required evidence | Tests |
| --- | --- | --- |
| Type graphs | Cycles, query order, correct results, terminating real causes. | [Prover](../../internal/checker/prover_test.go), [classifier](../../internal/checker/classification_test.go). |
| Source matcher | Complete accepted forms and rejected nearby forms. | [Shape checks](../../pkg/analyzer/shape_convergence_test.go), [CLI fixtures](../../tests/cli_test.go). |
| Builder state | Aliases, evaluation order, branch joins, loop resets. | [Freeze](../../internal/freeze/freeze_test.go). |
| Generation | Repeatable output, provisional names, stale-file rejection. | [Generator](../../internal/projectiongen/generate_test.go), [CLI generation](../../tests/generation_test.go). |
| Frozen pointer storage | Ordinary handles, protected targets, nil, selected identity, child reads, pooled reuse, and allocation costs. | [CLI and vet](../../tests/frozen_pointer_storage_test.go), [runtime cases](../../tests/testdata/frozen_pointer_storage/valid/selection_test.go). |
| Runtime storage | Shared Freeze storage, independent Clone storage, publication, allocations. | [Slice](../../pkg/im/slice_test.go), [Cell](../../pkg/im/cell_test.go). |
| Service lifetime | Setup, publication, retries, child cleanup, list removal. | [Init](../../pkg/supervise/init_test.go), [runtime](../../pkg/supervise/supervise_test.go), [registry](../../pkg/supervise/registry_test.go). |
| Import boundary | Original failure survives wrappers and both drivers. | [Dependency graph](../../pkg/analyzer/dependencies_graph_test.go), [consumer boundary](../../tests/boundary_test.go). |

## CLI evidence

```text
source + separate generated Go files
    → independent consumer module, GOWORK=off
    → built command or vet driver
    → combined stdout and stderr
    → normalize paths; sort complete diagnostic records
    → one output.txt per fixture package
```

A snapshot records one program's result. The [review requirements](../reviewer-focus.md)
also require tracing each allowed form through the reference, complete matcher,
ownership checks, and accepted/rejected fixtures.

[Hosted CI](../../.github/workflows/ci.yml) runs source and generated-document checks,
repository analysis, and race tests on Linux, macOS, and Windows. The
[development guide](../development.md) contains commands and snapshot procedures.
