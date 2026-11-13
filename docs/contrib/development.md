# Development

This single-module Go monorepo follows the scaffolding in `../tgo`: a thin
command, public packages, internal analysis, separate end-to-end tests, pinned
lint tools, generated package documentation, and hosted CI.

```sh
make build
make test-unit
make test-e2e
make lint
make docs
make docs-check
```

Run focused tests locally. Push commits to run the full race-enabled suite on
Linux, macOS, and Windows. CI also checks source, generated documentation, the
CLI build, and the repository with its own analyzer. Workflows run on pushes
and pull requests.

`cmd/immutago` calls `pkg/analyzer`, which uses `internal/checker`.
`pkg/immutable` provides the marker and value collections. Internal packages
must not import the analyzer or command. The library's public APIs need not be
used by the CLI; dead-code checks include tests.

Every named handwritten function needs a comment. Files have a 700-line limit;
functions have a 100-line limit; lines have a 120-character limit. Lint includes
vet, Staticcheck, Revive, complexity, dependency, and formatting checks.
Fixtures under `testdata` are exempt from function-comment and style checks.

Edit package Go comments, then run `make docs`. Commit the generated `readme.md`
files with their source comments. The gomarkdoc templates and source checks
come from TGo; its compiler patches and language fixtures are not copied.
Use the [ADR template](../adr/template/what.md) for substantial design decisions.
