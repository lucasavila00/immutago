# Repository checks

`make lint` runs these checks and golangci-lint:

| Tool | Check |
| --- | --- |
| [check-source-size.go](check-source-size.go) | Tracked Go files have at most 700 lines. |
| [check-function-comments](check-function-comments/main.go) | Named handwritten functions have comments; generated files and testdata are excluded. |
| [check-dead-code.sh](check-dead-code.sh) | The pinned x/tools deadcode tool finds no unreachable code, with tests as entry points. |

[gomarkdoc templates](gomarkdoc/package.gotxt) generate package references from
Go comments. [.gomarkdoc.yml](../.gomarkdoc.yml) selects the templates;
`make docs` writes the references and `make docs-check` checks them.

[dependency-stress](dependency-stress/main.go) writes the dense package graph
from issue 20. Use `go run ./scripts/dependency-stress -scale 5 -out /tmp/deps`
for 400 ordinary packages, which should pass under ordinary Go API trust. Add
`-library /absolute/path/to/immutago` to give every generated package a direct
`im` import and add a consumer. Those packages remain checked: local unsafe
errors must be reported, and the consumer receives one representative original
failure. The
[dependency workflow](../.github/workflows/dependency-performance.yml) runs it
through both analysis drivers and saves measurements.
