# Development

[Documentation](README.md) · [Architecture](arch/README.md) · [Review](reviewer-focus.md)

Use the Go version in [go.mod](../go.mod). Run repository commands from the root.
Read the [rules](rules.md), then change the matcher, proof checks, documentation,
and accepted/rejected tests together. Each new form needs a rejecting fallback.

## Validation

Use local runs for focused debugging. Push the branch and use
[GitHub Actions](../.github/workflows/ci.yml) for full validation.

| Command | Purpose |
| --- | --- |
| `make build` | Build `bin/immutago`. |
| `go test ./internal/freeze -run TestFreezeOwnership -count=1` | Debug builder ownership. |
| `go test ./tests -run '^TestCLI/documentation$' -count=1` | Check guide fixtures through the CLI. |
| `make docs` | Generate package references from Go comments. |
| `make docs-check` | Check generated references. |
| `make bench` | Measure collection costs. |
| `make bench-core` | Measure core scaling; CI also saves profiles. |

CI runs the broad test, lint, build, generated-doc, and self-analysis checks.
Race-enabled tests run on Linux, macOS, and Windows.

## Diagnostic snapshots

Each CLI source has a sibling `.txt` file with exact diagnostics. Inspect output
before updating it. For one changed fixture group:

```sh
go test ./tests -run '^TestCLI/documentation$' -count=1
go test ./tests -run '^TestCLI/documentation$' -count=1 -update-snapshots
git diff -- tests/testdata
```

A passing snapshot is not proof of the rule. Check the accepted forms and nearby
rejections. Consumer-only dependency cases must also pass through CLI and vet.

Check changed documentation links and run complete examples through the CLI.
Keep language rules in the guide and implementation facts in architecture.
[Source checks](../scripts/README.md) define comment and size requirements.
