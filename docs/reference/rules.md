# Checker reference

Use this reference to check which Go forms Immutago accepts. The
[glossary](glossary.md) defines its terms.

## Immutable data forms (`im`)

| Reference | Use it to |
| --- | --- |
| [Type rules](types.md) | Check the shared Frozen, Copied, and read-result categories. |
| [Common forms](common-forms.md) | Check stored types, construction, reads, pointers, calls, and statements. |
| [Frozen](frozen-records.md) | Store or return pointers to immutable objects. |
| [Frozen interfaces](frozen-interfaces.md) | Store proved implementations and call through local projections. |
| [Local](local.md) | Combine local projections with copied values. |
| [Copied](copied.md) | Return value records and change independent copies. |
| [Cell](cell.md) | Publish a Frozen value and read its current projection. |
| [Typed projections](projections.md) | Generate field projections and choose explicit storage operations. |
| [Projection primitives](projection-primitives.md) | Check the fixed projection APIs. |
| [Frozen values](frozen.md) | Store strings, slices, and maps; read or explicitly clone them. |
| [Opaque handles](unchecked-values.md) | Keep a trusted pointer behind a package API. |

## Service forms (`supervise`)

- [Service results](service-results.md): result types for exported service methods.
- [Service references](service-references.md): stored handles and stable service pointers.
- [Supervised services](services.md): execution, failure, and shutdown.

## Rules for im and supervise

Use im type rules to check immutable data. Review service state changes and
synchronization in the service code.

Use the common forms with each type's forms. **Exhaustive** means all allowed
forms are listed. Unlisted operations on protected storage are rejected.
Service-reference values use ordinary Go storage and transport. Their fields and
pointer construction keep the [Frozen record rules](service-references.md#checker-rules--exhaustive).
Borrow results use ordinary Go rules.

### Required generation

Declare Frozen records other than opaque handles, and Frozen interfaces, as named
types at package scope in non-test Go files. Every package with these declarations
needs one exact directive:

```go
//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen
```

| Package setup — exhaustive | Requirement |
| --- | --- |
| Frozen record other than an opaque handle, or Frozen interface declaration | Named package-scope type in a `.go` file that does not end in `_test.go`. |
| Directive | Use exactly one copy of the command above, at column 1 in a non-test package source file. |
| Output | Keep `zz_immutago.gen.go` from the current declarations and build configuration. |
| Frozen records other than opaque handles | Generate typed field projections, Projection constructors, and Cells. |
| Frozen interfaces | Generate local method projections and Projection constructors. |

Run `go generate ./...` after a record or interface changes. If no declarations that require generation
remain, either keep the directive and generate once more to write an empty package
file, or remove both the directive and `zz_immutago.gen.go`. You can remove the
directive after generation writes the empty file. If the directive is already gone,
remove the stale output or run `go run github.com/lucasavila00/immutago/cmd/immutago-gen`
in that package.

Generation uses the active `GOOS`, `GOARCH`, and build tags. Build constraints can
select Frozen declarations; the generated file has no constraint. Use the same
configuration for generation, the checker, and the build.

| Configurations | Required workflow |
| --- | --- |
| Same generated declarations, field types, array lengths, and interface methods | Keep one output file if all configurations generate the same bytes. Constrained files can use its projections and Cells. |
| Different generated output | Regenerate before each configuration's check and build. Use separate checkouts or build directories for concurrent configurations. |
| A configuration excludes all declarations that require generation | Keep the directive in an unconstrained file, or use the direct generator command above to replace stale output with an empty package file. |
| Imported Frozen types | Generate their owning packages for the same configuration. `go generate ./...` does not generate dependencies outside the selected packages. |

The checker rejects a missing directive, missing output, or stale output before
source checks. Only verified generated code can form interior Frozen-record addresses.
Projection.Unchecked explicitly returns a Frozen payload for shared use. All normal
type, write, capture, generic, receiver, and transport checks still apply.
Opaque handles, Local records, and Copied records need no generation.
Named opaque handles can be declared locally or in test files.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| A Frozen record other than an opaque handle with no generation directive | Add the exact directive and run generation. |
| A Frozen record other than a named opaque handle inside a function or `_test.go` file | Move its declaration to a non-test Go file at package scope. |
| An anonymous Frozen record, including a field type | Declare a named package-scope record and use that type. |
| An alternate generation command in the directive | Use the canonical command above. |
| A manually changed or stale generated file | Regenerate from the source declarations with the same build configuration. |
| Output made with `GOFLAGS=-tags=audit`, checked with different schema types and no tag | Regenerate without the tag, or keep the schema and its type dependencies the same in both configurations. |

### Opaque handles

[UncheckedValue and its Frozen wrappers](unchecked-values.md) keep trusted pointers.
Their target can be mutable. The declaring package controls access and synchronization.
The checker proves the wrapper fields and rejects Local payloads.

### Check the program

Run `immutago ./...` or `go vet -vettool=/absolute/path/to/immutago ./...`.
`go build` alone does not enforce these rules.

For services, also run standard `go vet ./...` to check common copies of Service
and Group owners. Immutago does not enforce their
[Go copy contract](services.md#copy-contract), and a custom vettool does not run
the standard vet checks.

For build tags, use `GOFLAGS=-tags=integration immutago ./...` or
`go vet -vettool=/absolute/path/to/immutago -tags=integration ./...`.
The CLI's own `-tags` flag does not select tagged files. See
[Check in CI](../guides/getting-started.md#check-in-ci).

### Read diagnostics

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

### Package boundary

The checker checks a package in any of these cases, including its test files:

| Package checks — exhaustive | Example |
| --- | --- |
| Imports `im` or `supervise`, including an alias or blank import | `import _ "github.com/lucasavila00/immutago/pkg/im"` |
| Uses an im or supervise type, including inside another type | `var point api.Point`, where Point embeds `im.Frozen` |
| Uses a function, interface, or generic type that mentions an im or supervise type | `callback := api.Publish`, where Publish takes `*api.Point`; `api.Alias[int]`, where Alias has a Cell type in its constraint |
| Imports a package that failed its checks, directly or through another package | Import a package with an invalid immutable field. |

- Aliases, type arguments, and generic constraints also count.
- A package that uses only supervise receives service type and result checks.
  Using im types, including the Frozen fields of Ref, also selects the im source checks.
- The CLI and `go vet` use the same rules.
- Other packages are trusted to obey Go contracts, including immutable strings.
- The `im`, `supervise`, and standard-library implementations are trusted.
- The generated Go test runner is skipped; user tests are checked.
- User code has no suppression comments. `im.UncheckedValue` delegates
  payload access and synchronization to its declaring package.
- Importing `unsafe` or `reflect` is allowed; ordinary values use ordinary Go rules.
- Protected object values need a listed form: `reflect.ValueOf(1)` is allowed,
  but `reflect.ValueOf(local)` with a Local record value fails the interface-boxing rule.
  Frozen values, pointer handles, and their ordinary carriers use the listed Go forms.
  Callers must not give a Frozen pointer to an API that changes its target;
  mutation effects through ordinary interfaces, reflection, or unsafe code are not proved.

| Rejected examples — non-exhaustive | Use instead |
| --- | --- |
| Cgo in a package that imports or uses im | Ordinary typed Go code. Cgo produces one error at the source package line. |
| A function without a body whose signature contains an im type other than Copied or an opaque handle | Supply a Go body to check its protected operations. |

An imported package's error includes one cause and a command to see its other errors.

Functions without a body use ordinary Go rules when their compiler-resolved
receiver, parameters, results, and generic constraints contain no im type other
than Copied or an opaque handle. Their ordinary holders can also use functions
without a Go body. For example, `func cpuID() uint64` and
`func checksum(data []byte) uint64` can use assembly implementations in a checked
package. A declaration such as `func read(value *Config) int`, where Config embeds
im.Frozen, needs a Go body.
