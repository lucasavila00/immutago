# Source checks

Adapted from TGo. `check-source-size.go` checks tracked Go files (700 lines).
`check-function-comments` checks comments on named handwritten functions,
excluding testdata and generated files. Both have tests.

`check-dead-code.sh` uses the pinned x/tools deadcode command with test entry
points. Public library APIs are exercised by tests rather than required to be
reachable from the CLI. `make lint` runs these checks and golangci-lint.

`gomarkdoc/` contains the package documentation templates selected by
[.gomarkdoc.yml](../.gomarkdoc.yml). `make docs` writes each package's generated
`readme.md`; `make docs-check` checks that it matches the Go comments.
