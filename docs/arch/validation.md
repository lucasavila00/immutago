# Validation

## Trust boundary

| Trusted source | Scope |
| --- | --- |
| Go compiler, runtime, and standard library | Compiler types, runtime behavior, and standard APIs. |
| Fixed im and supervise implementations | Collection storage, Owner, and service runtime. |
| Ordinary packages outside the [package boundary](../rules.md#package-boundary) | Go API contracts, including immutable strings and no hidden writes to protected memory. |
| `im.Trust` | The declaring package's immutability promise. |
| `supervise.AssumeConcurrentSafe` | The caller's concurrency promise. |

[Toolchain detection](../../pkg/analyzer/toolchain.go):

- Recognizes standard-library source under the active `go env GOROOT`.
- Skips generated `.test` main packages with the compiler-checked standard
  internal test dependency.
- Still checks user test packages.

Enforcement:

- Run the analyzer for each build configuration; `go build` adds no checks.
- Checked packages currently reject unsafe/reflect imports, cgo, and bodyless functions.
- There are no suppression directives. The two [trust markers](../rules.md#trust-markers)
  above are the explicit user promises.
- Checked dependency failures reach consumers through package facts.
- Freeze has no runtime fallback when the analyzer is omitted.

## Test map

| Evidence | Source |
| --- | --- |
| Concrete recursive type proofs and marker modes | [Type tests](../../internal/checker/types_test.go), [marker tests](../../internal/checker/markers_test.go) |
| Immutable writes and pointer escape rules | [Object tests](../../pkg/analyzer/immutable_objects_test.go), [escape tests](../../pkg/analyzer/escape_test.go) |
| Complete allowed shapes and nearby rejections | [Subset tests](../../pkg/analyzer/explicit_subset_test.go), [shape tests](../../pkg/analyzer/shape_convergence_test.go) |
| Builder aliases, loop resets, and consumption | [Freeze tests](../../internal/freeze/freeze_test.go) |
| Storage reuse, clone independence, identity, and allocation counts | [Slice tests](../../pkg/im/slice_test.go), [map tests](../../pkg/im/map_test.go), [iterator tests](../../pkg/im/iterator_test.go) |
| Service declarations, Refs, receiver uses, and start arguments | [Service tests](../../pkg/analyzer/service_test.go), [dependency tests](../../pkg/analyzer/service_dependency_test.go), [marker tests](../../internal/checker/markers_test.go), [CLI fixtures](../../tests/testdata/accepted/service) |
| Init signatures, startup errors, publication, and failed setup cleanup | [Checker tests](../../pkg/analyzer/service_init_test.go), [runtime tests](../../pkg/supervise/init_test.go), [CLI example](../../tests/testdata/accepted/serviceinit/main.go) |
| Service guide startup and background file refresh | [Guide program](../../tests/testdata/accepted/servicedoc/main.go), [refresh test](../../tests/testdata/accepted/servicedoc/main_test.go), [child-pool fixture](../../tests/testdata/accepted/service/pool_init.go) |
| UncheckedValue handles: declaring-package wrap and read, and storage as an allowed type | [UncheckedValue tests](../../pkg/analyzer/unchecked_value_test.go), [CLI fixtures](../../tests/testdata/accepted/uncheckedvalue) |
| Restart, backoff, escalation, stop order, reserved Refs, and the test helper | [Runtime tests](../../pkg/supervise/supervise_test.go), [helper tests](../../pkg/supervise/supervisetest/supervisetest_test.go) |
| Consumer-only dependency failures | [Dependency tests](../../pkg/analyzer/dependencies_test.go), [graph and gob tests](../../pkg/analyzer/dependencies_graph_test.go), [CLI fixtures](../../tests/cli_test.go) |

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
