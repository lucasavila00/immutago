# Validation and trust boundary

The analyzer's guarantee depends on checked source forms and reviewed runtime
contracts. Tests must check both: a matcher can accept an unsafe operation even
when the runtime behaves correctly, and a runtime can copy or expose storage
even when a matcher accepts only the intended calls.

## What the implementation trusts

The compiler supplies resolved types, object identities, method selections, and
valid Go syntax. The Go runtime and standard library supply their documented
behavior. The fixed im and supervise packages are trusted implementations:
the analyzer skips their source and checks callers against their contracts.

The projection generator has two specific permissions in its exact verified
output: calling Projection.Read and taking interior Frozen-record addresses.
The analyzer computes expected output and checks the physical token file before
it grants those permissions. It still checks the generated file's normal types,
writes, captures, receivers, generics, and transport.

Ordinary APIs outside protected operations must obey Go contracts. For example,
an ordinary string helper must not change the bytes behind a string. The
analyzer does not prove scalar origins, all unsafe memory effects, or service
method effects. Reflection and unsafe calls that involve protected storage must
pass the normal source matcher; they do not get a general exemption.

`im.UncheckedValue[T]` is an explicit package promise. Its declaring package
uses Trust to wrap the value and Unwrap to access it, and promises that the
payload never changes. The checker treats the handle as opaque; it does not
prove that promise. There are no suppression directives.

## Where checks run

A normal Go build does not run immutago. Run the analyzer for each relevant
build configuration, including its tags, target, and selected files. Freeze
has no runtime alias check or consume flag if the analyzer is omitted.

Direct imports and used API types select packages. A failed checked dependency
also selects its consumers through package facts. Packages that use only
supervise get service result and reference checks. Packages that import or use
im also get type, generated-file, source, and builder checks. The full selection
algorithm is in [package_scope.go](../../pkg/analyzer/package_scope.go).

[Toolchain detection](../../pkg/analyzer/toolchain.go) skips standard-library
files under the active toolchain's GOROOT source directory and generated test
main packages identified by their internal test dependency. It does not skip
user tests. In im packages, cgo is rejected, and functions with protected im
signatures require Go bodies. Ordinary Copied signatures and service references
keep Go rules. These restrictions make relevant operations visible to source
checks; they do not claim to inspect every dependency effect.

## From a defect to regression evidence

Start with a small program that reproduces the wrong decision. Identify the
missing type classification, incomplete shape match, or incorrect builder
state. Test the class of operation, including nearby declarations, nil or zero
forms, aliases, nested storage, tuples, and variadic arguments where relevant.
An accepted example needs a corresponding allowed form in the language
reference. A rejected example needs an ordinary replacement when one exists.

Use the test layer that can expose the defect, then keep end-to-end evidence:

| Layer | What it establishes | Useful source |
| --- | --- | --- |
| Type proof | Recursive category acceptance, cycles, query order, and a terminating real failure cause. | [Prover tests](../../internal/checker/prover_test.go), [category tests](../../internal/checker/categories_test.go). |
| Source matching | Exact accepted shapes and rejection of nearby shapes, including implicit receiver operations. | [Subset tests](../../pkg/analyzer/explicit_subset_test.go), [shape tests](../../pkg/analyzer/shape_convergence_test.go). |
| Builder analysis | Exclusive handle use, evaluation order, branch joins, loop resets, and consumption. | [Freeze tests](../../internal/freeze/freeze_test.go). |
| Generation | Fresh generation, stale-file rejection, bootstrap type names, and repeatable compiler-checked output. | [Generator tests](../../internal/projectiongen/generate_test.go), [CLI generation tests](../../tests/generation_test.go). |
| Dependency facts | Original-cause selection, forwarding through wrappers, and exact serialization. | [Graph tests](../../pkg/analyzer/dependencies_graph_test.go). |
| Runtime data | Shared storage after Freeze, independent Clone slots, atomic publication, representation, and allocation counts. | [Slice tests](../../pkg/im/slice_test.go), [map tests](../../pkg/im/map_test.go), [Cell tests](../../pkg/im/cell_test.go). |
| Service runtime | Initial setup failures, publication, independent retries, child cleanup, stop order, and registry removal. | [Init tests](../../pkg/supervise/init_test.go), [runtime tests](../../pkg/supervise/supervise_test.go), [registry tests](../../pkg/supervise/registry_test.go). |
| Real consumer | Driver behavior, selected packages, source positions, diagnostic text, JSON, and vet facts. | [CLI harness](../../tests/cli_test.go), [boundary tests](../../tests/boundary_test.go). |

[Classification tests](../../internal/checker/classification_test.go) and
[scaling tests](../../pkg/analyzer/type_scaling_test.go) compare recursive walks
with independent results. Runtime examples such as [shipping quotes](../../examples/shipping/quote_test.go)
and [search queries](../../examples/search/query_test.go) check application
behavior beyond isolated operations. No test layer alone establishes the full
supported-language contract.

### CLI fixtures and snapshots

The CLI harness builds the shipped command, copies fixtures into an independent
consumer module, and resolves the library to the checkout. It disables workspace
mode. Tests run accepted and rejected programs through the command and vet
driver, including consumers selected without directly running the failed
origin package.

Each fixture package has one `output.txt` for combined stdout and stderr. The
harness normalizes machine paths and line endings, then sorts complete
diagnostic records without separating their context lines. Source files and
`zz_immutago.gen.go` stay separate. A compiler failure is distinct from an
analyzer rejection; the harness also checks invalid Go and JSON output.

Inspect changed diagnostics before updating a snapshot. For one fixture group:

```sh
go test ./tests -run '^TestCLI/documentation$' -count=1 -update-snapshots
```

Run the same target without the update flag to verify the saved result. Keep
the target as narrow as the failure permits. [The development guide](../development.md#diagnostic-snapshots)
describes the snapshot files and commands. A snapshot proves that the saved
program produced the saved result; it does not prove that all forbidden forms
are rejected.

## Hosted validation

[CI](../../.github/workflows/ci.yml) runs on pull requests and main pushes.
The Linux checks job runs source checks, lint and formatting, generated reference
checks, command builds, and repository self-analysis. Test jobs run
`go test -race -count=1 ./...` on Linux, macOS, and Windows. `make docs-check`
checks generated API references; changed prose and local links also need review.
Benchmarks run separately and are not part of routine CI.

Use local tests for a specific failure and hosted CI for full validation. Before
accepting a checker change, trace each changed allowed form through its complete
reference entry, matcher, ownership checks, accepted fixture, rejected nearby
forms, and diagnostic snapshot. A forbidden form that passes is a defect even
if no existing rejection test lists it. Tests are regression evidence, not a
formal proof of the checker.
