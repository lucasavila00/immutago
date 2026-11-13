# End-to-end tests

`go test ./tests -count=1` builds the real `immutago` command once. It copies
fixtures into temporary consumer modules with a local module replacement and
workspace mode disabled. No fixtures depend on downloaded modules.

`testdata/accepted` covers construction, final-use borrow lifetimes, disjoint
fields, branches, loops, readable pointers, generic and cross-package alias
summaries, closures, collection copies, and recursive types. The rejected
fixtures pair these with overlapping writes while a loan remains live and writes
through read-only references. Structural field, marker, ownership, reflection,
unsafe, and native-declaration checks remain covered.

Every `// want "regex"` comment requires a diagnostic on the operation's exact
source line. Accepted fixtures must produce no diagnostics; a blanket ban on
pointers or local field writes cannot satisfy these tests.

Add accepted programs under `accepted`. Add a package under `rejected` for
negative cases; keep each source filename unique. Negative fixtures must be
valid Go so the test reaches the analyzer. Keep the raw-string import fixture
unformatted: gofmt changes it to an interpreted string. The bodyless function
fixture has an empty assembly file so Go can load its declaration before the
analyzer rejects it. The harness also checks `go vet`
integration, JSON output, and Go compiler error reporting.

Run focused tests locally. Hosted CI runs these tests with the race detector on
Linux, macOS, and Windows.
