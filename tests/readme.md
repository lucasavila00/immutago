# End-to-end tests

`go test ./tests -count=1` builds the real `immutago` command once. It copies
fixtures into temporary consumer modules with a local module replacement and
workspace mode disabled. No fixtures depend on downloaded modules.

`testdata/accepted` covers owned construction, value copies, final-use borrow
lifetimes, branches, loops, readable pointer parameters, direct cross-package
calls, scalar collection copies, and recursive immutable types. Core rejected
fixtures require diagnostics at writes through shared references or writes to
an owner with a live loan. Boundary fixtures reject reference returns, generic
borrowing, captures, interior scalar addresses, and native protected containers.
Structural field, marker, ownership, reflection, unsafe, and native-declaration
checks remain covered. Freeze fixtures build exclusive native slices and maps,
consume them with FreezeSlice or FreezeMap, and use Clone to make independent
mutable copies. Negative fixtures cover use after transfer, aliases, escapes,
branch joins, and loops. The runtime fixture checks the frozen values before
and after edits to their clones.

The explicit subset fixtures check fixed local references, named readers, and
reference reads through type aliases and parentheses. They reject reference
copies and assignments, user-defined methods (including promoted methods),
multiple assignments with protected storage, and nested Freeze calls. Select
fixtures reject protected communication clauses and permit checked case bodies.
They include a receive that writes its second result into a consumed builder.

Each source file has an adjacent `.txt` snapshot, as in TGo. Snapshots preserve
exact diagnostic messages, source lines, and columns. Accepted sources have
empty snapshots. CLI and `go vet` must produce the same per-source diagnostics.
The harness removes machine paths and driver headings, then sorts complete diagnostic records. Indented context and related source locations stay with their primary
diagnostic. Compiler load errors have their own source and text fixture.

After an intentional diagnostic change, run:

```sh
go test ./tests -run '^TestCLI$' -update-snapshots
```

Review each changed `.txt` file before committing. Snapshot updates are explicit;
a normal test run never changes them.

Add accepted programs under `accepted`. Add a package under `rejected` for
negative cases; keep each source filename unique. Negative fixtures must be
valid Go so the test reaches the analyzer. Keep the raw-string import fixture
unformatted: gofmt changes it to an interpreted string. The bodyless function
fixture has an empty assembly file so Go can load its declaration before the
analyzer rejects it. The harness also checks `go vet`
integration, JSON output, and Go compiler error reporting. Dependency fixtures
run as a batch and as selected consumers, so hidden dependency failures must
propagate through import chains.

Run focused tests locally. Hosted CI runs these tests with the race detector on
Linux, macOS, and Windows.
