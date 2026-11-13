# End-to-end tests

`go test ./tests -count=1` builds the real `immutago` command once. It copies
fixtures into temporary consumer modules with a local module replacement and
workspace mode disabled. No fixtures depend on downloaded modules.

`testdata/accepted` covers immutable construction, scalar preparation, complete
variable rebinding, value copies, shared pointer copies and returns, stored
references, recursive pointer fields, and direct cross-package calls. Rejected
fixtures retain earlier mutable-owner programs: field writes fail even when
there is no live reference. Reference boundaries reject named-variable
addresses, interior scalar addresses, pointer indirection, generic protected
calls, captures, and unsupported interface transport.

Freeze fixtures build exclusive native slices and maps, consume them with
FreezeSlice or FreezeMap, and use Clone for independent mutable storage.
Negative cases cover aliases, escapes, uses after transfer, branches, and loops.
Statement tests cover one destination, checked select bodies, fresh Get results,
and direct Freeze assignments. They reject nested Freeze and protected select
communication operands. Runtime tests check that edits to clones leave frozen
storage unchanged.

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
