# Commands and diagnostics

## Check packages

Run the checker from a Go module:

```sh
immutago ./...
```

Or use it with go vet:

```sh
go vet -vettool=/absolute/path/to/immutago ./...
```

Both commands use the same [package selection rules](../../docs/reference/rules.md#package-boundary).
`go build` alone does not enforce im rules. Errors make the normal checker command
fail. Do not use `-json` as a CI pass/fail check: it can return exit code 0 with errors.

For services, also run standard `go vet ./...` to check common copies of Service
and Group values. A custom vettool does not run the standard vet checks.
Immutago does not enforce the [copy contract](../../pkg/supervise/contract.md#copy-contract).

## Build configurations

Generate and check each target platform and set of build tags. Set `GOFLAGS`
for the CLI, or pass `-tags` to go vet:

```sh
GOFLAGS=-tags=integration go generate ./...
GOFLAGS=-tags=integration immutago ./...
go vet -vettool=/absolute/path/to/immutago -tags=integration ./...
```

`immutago -tags integration ./...` does not select tagged files. Use one of the
commands above, including for packages whose files all require a tag.

One generated file can serve configurations that produce the same bytes.
Otherwise, regenerate before each configuration's check and build. Use separate
checkouts or build directories for concurrent configurations with different
output. Generate imported Frozen types in their owning packages for the same
configuration; `go generate ./...` does not generate dependencies outside the
selected packages. See the [generation requirements](../../docs/reference/rules.md#required-generation).

For a cross build of one package, build the tools for the host before selecting
the target:

```sh
go build -o ./bin/immutago-gen github.com/lucasavila00/immutago/cmd/immutago-gen
go build -o ./bin/immutago github.com/lucasavila00/immutago/cmd/immutago
GOOS=windows GOARCH=amd64 ./bin/immutago-gen
GOOS=windows GOARCH=amd64 ./bin/immutago .
```

Do not use the go run generation directive to execute a cross-built tool.

## Update generated output

Run `go generate ./...` after Frozen records or interfaces change, and after an
upgrade to the Immutago library or tools. Keep generated output with the source.
Go type checking runs first. If stale output does not compile, the command reports
Go type errors; regenerate, then run the checker again.

After the last declaration that requires generation is removed, either remove
both the directive and `zz_immutago.gen.go`, or keep the directive and generate
once more to write an empty package file. The directive can then be removed.
If the directive is already gone or inactive under the selected build tags,
remove stale output when none is required, or run the generator directly in that
package:

```sh
go run github.com/lucasavila00/immutago/cmd/immutago-gen
```

When a configuration still has declarations that require generation, keep the
exact required directive in an active source file.

## Read diagnostics

A diagnostic can include these context fields:

| Line | Meaning |
| --- | --- |
| `source:` | The source expression or statement for the rejected operation. |
| `type:` | The compiler-resolved expression type, or the type checked by a type or package rule. |
| `owner:` | The record that declares the field. |
| `field:` | The record and field name for a declaration error. |
| `field type:` | The compiler-resolved field type, on the same line as its owner. |
| `result type:` | The compiler-resolved service method result type. |
| `rule:` | The rule that failed. |
| `representative failure:` | One original diagnostic from a failed dependency, with its location and indented context lines. |
| `note:` | Other errors can exist in that dependency. |
| `fix:` | A supported replacement for the failed rule. |

If `Ptr` returns `int64`, `source: r.C.Ptr()` has `type: int64`, even when the
rule rejects the receiver operation. A method value such as `h.Ptr` has type
`func() int64`.
