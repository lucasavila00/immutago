# CLI tests

[cli_test.go](cli_test.go) builds the real command and copies fixtures into
independent consumer modules with a local module replacement. It checks CLI,
vet, JSON, compiler errors, dependency failures, and runtime output.

Accepted fixtures cover the [allowed forms](../docs/reference/rules.md). Rejected fixtures
cover nearby unsupported forms. Each fixture package has one `output.txt` snapshot for combined stdout and
stderr; accepted snapshots are empty. The harness removes machine paths and
driver headings, then sorts complete messages without separating context lines.

Consumer-only dependency tests require failures to propagate through imports,
including wrappers. Runtime tests check storage transfer, Clone independence,
and unchanged shared objects after pointer rebinding.

[Copied record tests](copied_records_test.go) use both drivers to check fields,
decoders, ordinary mutation, pointers, callbacks, tuples, service results, generated
getters, and consumer-only failures. Nearby rejected forms check protected owners
and builder aliases before Freeze.
The accepted consumer tests also run with the race detector. The
[runtime cost tests](../pkg/im/copied_test.go) check record layout, zero-allocation
Value, clone independence, and native/projection copy costs.

The [file-cache fixture](testdata/accepted/servicedoc/main.go) runs through
the CLI and vet in `TestCLI/service_guide`. That test also checks its output
and runs the [background refresh test](testdata/accepted/servicedoc/main_test.go)
with the race detector. The [application examples](../examples/README.md) have
separate tests for their own behavior. Repository CI checks their source through
the CLI and runs their Go tests.

[Test-main tests](testmain_test.go) check generated runners and real internal and
external test files through the CLI and vet. A forged generated header is checked.

[Cgo tests](cgo_test.go) check real compiler-generated wrappers through CLI and
vet, including consumer-only reports for a cgo wrapper and its invalid dependency.
They run when cgo is enabled.

For a changed group:

```sh
go test ./tests -run '^TestCLI/documentation$' -count=1
go test ./tests -run '^TestCLI/documentation$' -count=1 -update-snapshots
```

Inspect each changed snapshot against its source and the rules. A snapshot
update does not prove correctness. Use hosted CI for full validation.

Keep fixture filenames unique and rejected source valid Go. Each rejection must
test its intended rule, not an unrelated invalid constructor. Do not format the
raw-string import fixture: gofmt changes its source form. The bodyless-function
fixture needs its empty assembly file so Go can load it before analyzer checks.

`TestServiceErrors` checks service error signatures, protected result/error
bindings, ordinary saved errors, and rejected signatures and Local uses through
the CLI and vet driver.

`TestCLI/proof_diagnostics` checks the proof category, failure cause, and repair
text for invalid fields, collections, projections, service results, builder
placement, service references, and direct calls. Its accepted fixture checks the
replacement forms. Both fixtures also run in the full CLI and vet checks.
