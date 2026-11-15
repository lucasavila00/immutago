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
| [Unchecked immutable values](unchecked-values.md) | Hold a value that the analyzer trusts and does not prove. |

## Service forms (`supervise`)

- [Service results](service-results.md): result types for exported service methods.
- [Service references](service-references.md): stored references and direct calls to the current service.
- [Supervised services](services.md): execution and restart behavior.

## Rules for im and supervise

Use im type rules to check immutable data. Review service state changes and
synchronization in the service code.

Use the common forms with each type's forms. **Exhaustive** means all allowed
forms are listed. Unlisted operations on protected storage and unlisted direct Borrow call forms
are rejected. Ordinary service-reference storage and transport use Go rules.

### Required generation

Declare Frozen records and Frozen interfaces as named types at package scope in
non-test Go files. Every package with these declarations needs one exact directive:

```go
//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen
```

| Package setup — exhaustive | Requirement |
| --- | --- |
| Frozen record or interface declaration | Named package-scope type in a `.go` file that does not end in `_test.go`. |
| Directive | Use exactly one copy of the command above, at column 1 in a non-test package source file. |
| Output | Keep `zz_immutago.gen.go` from the current declarations. |
| Frozen records | Generate typed field projections, Projection constructors, and Cells. |
| Frozen interfaces | Generate local method projections and Projection constructors. |

Run `go generate ./...` after a record or interface changes. If no Frozen declarations
remain, generation removes its output; the directive can stay. The checker rejects a missing
directive, missing output, or stale output. Only verified generated code can form interior Frozen-record addresses.
Projection.Unsafe explicitly returns a Frozen payload for shared use. All normal
type, write, capture, generic, receiver, and transport checks still apply.
Local and Copied records need no generation.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| A Frozen record with no generation directive | Add the exact directive and run generation. |
| A Frozen type inside a function or `_test.go` file | Move its declaration to a non-test Go file at package scope. |
| An anonymous Frozen record, including a field type | Declare a named package-scope record and use that type. |
| An alternate generation command in the directive | Use the canonical command above. |
| A manually changed or stale generated file | Regenerate from the source declarations. |

### Trusted immutable values

[`im.UncheckedValue[T]`](unchecked-values.md) records the declaring package's
promise that the wrapped value never changes. Only that package can wrap or read
it. The checker does not prove the promise.

### Check the program

Run `immutago ./...` or `go vet -vettool=/absolute/path/to/immutago ./...`.
`go build` alone does not enforce these rules.

### Package boundary

The checker checks a package in any of these cases, including its test files:

| Package checks — exhaustive | Example |
| --- | --- |
| Imports `im` or `supervise`, including an alias or blank import | `import _ "github.com/lucasavila00/immutago/pkg/im"` |
| Uses an im or supervise type, including inside another type | `var point api.Point`, where Point embeds `im.Frozen` |
| Uses a function, interface, or generic type that mentions an im or supervise type | `callback := api.Publish`, where Publish takes `*api.Point`; `api.Alias[int]`, where Alias has a Cell type in its constraint |
| Imports a package that failed its checks, directly or through another package | Import a package with an invalid immutable field. |

- Aliases, type arguments, and generic constraints also count.
- A package that uses only supervise receives service result and reference checks.
  Importing or using im also selects the immutable-source checks.
- The CLI and `go vet` use the same rules.
- Other packages are trusted to obey Go contracts, including immutable strings.
- The `im`, `supervise`, and standard-library implementations are trusted.
- The generated Go test runner is skipped; user tests are checked.
- User code has no suppression comments. `im.UncheckedValue` is the explicit
  immutability trust marker.
- Importing `unsafe` or `reflect` is allowed; ordinary values use ordinary Go rules.
- Protected object values need a listed form: `reflect.ValueOf(1)` is allowed,
  but `reflect.ValueOf(local)` with a Local record value fails the interface-boxing rule.
  Frozen values, pointer handles, and their ordinary carriers use the listed Go forms.
  Callers must not give a Frozen pointer to an API that changes its target;
  mutation effects through ordinary interfaces, reflection, or unsafe code are not proved.

| Rejected examples — non-exhaustive | Use instead |
| --- | --- |
| Cgo in a package that imports or uses im | Ordinary typed Go code. Cgo produces one error at the source package line. |
| A function without a body whose signature contains an im type other than Copied | Supply a Go body to check its protected operations. |

An imported package's error includes one cause and a command to see its other errors.

Functions without a body use ordinary Go rules when their compiler-resolved
receiver, parameters, results, and generic constraints contain no im type other
than Copied. Copied records and their ordinary holders can use functions without
a Go body. For example, `func cpuID() uint64` and
`func checksum(data []byte) uint64` can use assembly implementations in a checked
package. A declaration such as `func read(value *Config) int`, where Config embeds
im.Frozen, needs a Go body.
