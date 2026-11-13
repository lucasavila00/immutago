# Development

[Documentation](README.md) · [Architecture](arch/README.md) · [Review requirements](reviewer-focus.md)

## Set up and find the code

Use the Go version in [go.mod](../go.mod). This repository is one Go module.
Run commands from its root. Tool versions are pinned in `go.mod` and
[the Makefile](../Makefile); no separate installation of lint or documentation
tools is required for the default Make targets.

The [architecture code map](arch/README.md#code-map) identifies each component.
Internal packages must not import the analyzer or command. Public library APIs
need not be called by the CLI; dead-code checks include test entry points.

## Make a change

1. Read the [language rules](rules.md) and locate the matching source check.
2. Add or update an accepted case and a nearby rejected case for behavior changes.
3. Debug with the smallest relevant local test or analyzer run.
4. Inspect diagnostics and snapshot changes. Update the rules and architecture
   when the supported forms or implementation change.
5. Push the branch. Use hosted CI for full validation and complete the
   [review checklist](reviewer-focus.md).

An implementation mismatch is not fixed by rewriting only the docs. A new
allowed form needs its matcher, ownership checks, and tests in the same change.

## Commands

| Command | Purpose |
| --- | --- |
| `make build` | Build `bin/immutago`. |
| `go test ./internal/freeze -run TestFreezeOwnership -count=1` | Debug one ownership test group. |
| `go test ./tests -run '^TestCLI/review_convergence$' -count=1` | Run one CLI fixture group. |
| `make test-unit` | Run internal, public-package, and source-check unit tests. |
| `make test-e2e` | Run the end-to-end test package. |
| `make test` | Run the full race-enabled test suite. |
| `make lint` | Run source, dead-code, lint, and formatting checks. |
| `make docs` | Regenerate package documentation. |
| `make docs-check` | Check generated package documentation without changing it. |
| `make bench` | Measure collection operations and allocations. |

Use the broad test and lint targets in hosted CI for routine validation. Local
runs are for focused debugging and profiling. [CI](../.github/workflows/ci.yml)
runs on pushes and pull requests. It tests Linux, macOS, and Windows, and checks
source, generated docs, the CLI build, and repository self-analysis.

## Source and documentation conventions

Every named handwritten function needs a comment. Go files have a 700-line
limit; functions have a 100-line limit; lines have a 120-character limit.
Fixtures under `testdata` are exempt from function-comment and style checks.
See [source checks](../scripts/README.md) and [lint settings](../.golangci.yml).

Write plain, direct explanations and concrete Go examples. Keep allowed forms
complete; label rejected examples as non-exhaustive. The
[documentation index](README.md#where-documentation-belongs) assigns each topic
to its primary page. Check local links and run complete new examples through
the real CLI. Use Git history for earlier designs.

For generated references, edit Go comments and run `make docs`. Commit the
resulting `readme.md` files with their source comments. Templates live in
[`scripts/gomarkdoc`](../scripts/gomarkdoc), selected by
[.gomarkdoc.yml](../.gomarkdoc.yml).

## Diagnostic snapshots

CLI fixtures in `tests/testdata/` store exact compiler and analyzer output in
sibling `.txt` files. Snapshots include source positions, explanations, and
suggested corrections. Some dependency-driver snapshots are in
`tests/testdata/snapshots/`.

Run the affected subtest first. After checking that each output change is
correct, update that subtest's snapshots and review the diff:

```sh
go test ./tests -run '^TestCLI/review_convergence$' -count=1
go test ./tests -run '^TestCLI/review_convergence$' -count=1 -update-snapshots
git diff -- tests/testdata
go test ./tests -run '^TestCLI/review_convergence$' -count=1
```

Replace `review_convergence` with the subtest that covers your change. An
updated snapshot is not proof that the new behavior is correct. Check each
changed diagnostic against the guide and the source fixture. Use hosted CI
for the full suite.

See [collection costs](performance.md) for benchmark scope and allocation behavior.
