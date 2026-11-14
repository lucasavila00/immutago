# CLI tests

[cli_test.go](cli_test.go) builds the real command and copies fixtures into
independent consumer modules with a local module replacement. It checks CLI,
vet, JSON, compiler errors, dependency failures, and runtime output.

Accepted fixtures cover the [allowed forms](../docs/rules.md). Rejected fixtures
cover nearby unsupported forms. Each source has an exact sibling `.txt`
snapshot; accepted snapshots are empty. The harness removes machine paths and
driver headings, then sorts complete messages without separating context lines.

Consumer-only dependency tests require failures to propagate through imports,
including wrappers. Runtime tests check storage transfer, Clone independence,
and unchanged shared objects after pointer rebinding.

[Cgo tests](cgo_test.go) check real compiler-generated wrappers through CLI and
vet, including consumer-only dependency reports. They run when cgo is enabled.

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
