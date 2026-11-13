# End-to-end tests

`go test ./tests -count=1` builds the real `immutago` command once. It copies
fixtures into temporary consumer modules with a local module replacement and
workspace mode disabled. No fixtures depend on downloaded modules.

`testdata/accepted` covers cross-package values, aliases, generic constraints,
recursive collections, independent collection copies, and whole-value updates.
`testdata/rejected` covers invalid fields, mutable aliases, mutation, marker
placement, unsupported types, and unmarked wrappers. Every `// want "regex"`
comment requires a diagnostic on that exact source line.

Add accepted programs under `accepted`. Add a package under `rejected` for
negative cases; keep each source filename unique. Negative fixtures must be
valid Go so the test reaches the analyzer. The harness also checks `go vet`
integration, JSON output, and Go compiler error reporting.

Run focused tests locally. Hosted CI runs these tests with the race detector on
Linux, macOS, and Windows.
