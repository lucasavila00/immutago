# Validation

## Trust boundary

The checker trusts the Go compiler, runtime, standard library, and fixed `im`
and `proc` implementations. [Toolchain detection](../../pkg/analyzer/toolchain.go) identifies
standard library source under the active `go env GOROOT`. It identifies generated
Go test runners by their `.test` main package and compiler-checked standard
internal test dependency. It still checks user test packages. The analyzer does
not prove those toolchain implementations. Ordinary packages outside the
[package boundary](../rules.md#package-boundary) are also trusted to obey Go API
contracts, including immutable strings and no hidden writes to protected memory.
Their unsafe, reflection, and cgo implementations are not proved.

Application code must run through the analyzer for its selected build
configuration. An ordinary Go build supplies no immutability checks. Checked
code cannot use reflection, unsafe imports, or bodyless functions to bypass
the proof. There are no suppression directives. The one user trust point is
`proc.Trust`: a [process dependency](../processes.md#trusted-dependencies) is a
promise of safe concurrent use that the checker does not prove.
Failures from checked dependencies reach consumers through package facts. Runtime Freeze has
no fallback validation when the analyzer is omitted.

## Test map

| Evidence | Source |
| --- | --- |
| Concrete recursive type proofs and marker modes | [Type tests](../../internal/checker/types_test.go), [marker tests](../../internal/checker/markers_test.go) |
| Immutable writes and pointer escape rules | [Object tests](../../pkg/analyzer/immutable_objects_test.go), [escape tests](../../pkg/analyzer/escape_test.go) |
| Complete allowed shapes and nearby rejections | [Subset tests](../../pkg/analyzer/explicit_subset_test.go), [shape tests](../../pkg/analyzer/shape_convergence_test.go) |
| Builder aliases, loop resets, and consumption | [Freeze tests](../../internal/freeze/freeze_test.go) |
| Storage reuse, clone independence, identity, and allocation counts | [Slice tests](../../pkg/im/slice_test.go), [map tests](../../pkg/im/map_test.go), [iterator tests](../../pkg/im/iterator_test.go) |
| Process declarations, PIDs, receiver uses, and start arguments | [Process tests](../../pkg/analyzer/process_test.go), [dependency tests](../../pkg/analyzer/process_dependency_test.go), [marker tests](../../internal/checker/markers_test.go), [CLI fixtures](../../tests/testdata/accepted/process) |
| Restart, backoff, escalation, stop order, reserved PIDs, and the test helper | [Runtime tests](../../pkg/proc/proc_test.go), [helper tests](../../pkg/proc/proctest/proctest_test.go) |
| Consumer-only dependency failures | [UncheckedDependency tests](../../pkg/analyzer/dependencies_test.go), [graph and gob tests](../../pkg/analyzer/dependencies_graph_test.go), [CLI fixtures](../../tests/cli_test.go) |

The CLI harness builds the shipped command and loads independent consumer
modules. It checks diagnostics, source positions, JSON output, compiler errors,
and vet integration. Sibling `.txt` files record expected diagnostics; see the
[snapshot workflow](../development.md#diagnostic-snapshots).

## Hosted checks

[GitHub Actions](../../.github/workflows/ci.yml) runs lint and formatting,
generated documentation checks, the command build, and repository self-analysis
on Linux. It runs `go test -race -count=1 ./...` on Linux, macOS, and Windows.
Use local tests to debug specific failures and hosted CI for full validation.

These tests provide regression evidence, not a formal proof of the checker.
A supported form needs both a complete matcher and accepted/rejected test
coverage. [Collection benchmarks](../../pkg/im/benchmark_test.go) measure the
cost contract; the CI workflow does not run benchmarks.
