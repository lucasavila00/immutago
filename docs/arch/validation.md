# Validation

## Trust boundary

| Trusted source | Scope |
| --- | --- |
| Go compiler, runtime, and standard library | Compiler types, runtime behavior, and standard APIs. |
| Fixed im and supervise implementations | Collection storage, Cell, and service runtime. |
| Canonical projection generator | Read and interior Frozen-record addresses in its exact verified output; all normal source checks still apply. |
| Ordinary packages outside the [package boundary](../reference/rules.md#package-boundary) | Go API contracts, including immutable strings and no hidden writes to protected memory. |
| `im.Trust` | The declaring package's immutability promise. |

[Toolchain detection](../../pkg/analyzer/toolchain.go):

- Recognizes standard-library source under the active `go env GOROOT`.
- Skips generated `.test` main packages with the compiler-checked standard
  internal test dependency.
- Checks user test packages.

Enforcement:

- Run the analyzer for each build configuration; `go build` adds no checks.
- Unsafe and reflection operations use the normal source-form checks.
- Scalar origins and arbitrary unsafe memory effects are not proved; ordinary
  helpers must obey Go API contracts, including string immutability.
- Packages that import or use im reject cgo. Functions whose signatures contain
  im types or service references require Go bodies. Packages that use only supervise
  receive result and reference checks.
- There are no suppression directives. [UncheckedValue](../reference/unchecked-values.md)
  records the declaring package's immutability promise.
- Checked dependency failures reach consumers through package facts.
- Freeze has no runtime fallback when the analyzer is omitted.

## Test map

| Evidence | Source |
| --- | --- |
| Generator startup, repeatable output, Frozen records, and changed fields | [Generator tests](../../internal/projectiongen/generate_test.go) |
| Frozen, Copied, ReadResult, and recursive marker proofs | [Category tests](../../internal/checker/categories_test.go), [type tests](../../internal/checker/types_test.go), [marker tests](../../internal/checker/markers_test.go) |
| Copied declarations, decoders, ordinary mutation and calls, protected owners, service results, and dependency failures | [Copied proof tests](../../internal/checker/copied_test.go), [source checks](../../pkg/analyzer/copied_test.go), [ordinary Go checks](../../pkg/analyzer/copied_mutable_test.go), [CLI and vet tests](../../tests/copied_records_test.go) |
| Copied layout, allocation counts, and native/projection copy costs | [Runtime tests and benchmark](../../pkg/im/copied_test.go) |
| Immutable writes and pointer escape rules | [Object tests](../../pkg/analyzer/immutable_objects_test.go), [escape tests](../../pkg/analyzer/escape_test.go) |
| Complete allowed shapes and nearby rejections | [Subset tests](../../pkg/analyzer/explicit_subset_test.go), [shape tests](../../pkg/analyzer/shape_convergence_test.go) |
| Builder aliases, loop resets, and consumption | [Freeze tests](../../internal/freeze/freeze_test.go) |
| Storage reuse, clone independence, identity, and allocation counts | [Slice tests](../../pkg/im/slice_test.go), [map tests](../../pkg/im/map_test.go), [iterator tests](../../pkg/im/iterator_test.go) |
| Generated field reads without record copies | [Generated projection cost fixture](../../internal/projectiongen/testdata/readcost/README.md) |
| Service result types and ordinary service code | [Service tests](../../pkg/analyzer/service_test.go), [CLI fixtures](../../tests/testdata/accepted/service) |
| Service targets, direct Borrow call forms, and ordinary Go transport | [Reference tests](../../pkg/analyzer/service_handles_test.go), [CLI fixtures](../../tests/testdata/accepted/service/handles.go), [rejected forms](../../tests/testdata/rejected/service/handles.go) |
| Startup errors, publication, and failed setup cleanup | [runtime tests](../../pkg/supervise/init_test.go), [CLI fixture](../../tests/testdata/accepted/serviceinit/main.go) |
| Application calculations and reloads | [Shipping quotes](../../examples/shipping/quote_test.go), [catalog reloads](../../examples/shipping/loader_test.go), [search ranking](../../examples/search/query_test.go), [index updates](../../examples/search/indexer_test.go) |
| CLI source coverage for file refresh and child pools | [File-cache fixture](../../tests/testdata/accepted/servicedoc/main.go), [refresh test](../../tests/testdata/accepted/servicedoc/main_test.go) |
| UncheckedValue handles: declaring-package wrap and read, and storage as an allowed type | [UncheckedValue tests](../../pkg/analyzer/unchecked_value_test.go), [CLI fixtures](../../tests/testdata/accepted/uncheckedvalue) |
| Restart, backoff, escalation, stop order, reserved service references, and the test helper | [Runtime tests](../../pkg/supervise/supervise_test.go), [helper tests](../../pkg/supervise/supervisetest/supervisetest_test.go) |
| Consumer-only dependency failures | [Dependency tests](../../pkg/analyzer/dependencies_test.go), [graph and gob tests](../../pkg/analyzer/dependencies_graph_test.go), [CLI fixtures](../../tests/cli_test.go) |

The CLI harness builds the shipped command and loads independent consumer
modules. It checks diagnostics, source positions, JSON output, compiler errors,
and vet integration. Each fixture package has one `output.txt` for combined
stdout and stderr. Source and generated Go files stay separate; see the
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
