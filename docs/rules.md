# Checker reference

Use this reference to check which Go forms Immutago accepts. The
[glossary](glossary.md) defines its terms.

## Immutable data forms (`im`)

| Reference | Use it to |
| --- | --- |
| [Common forms](common-forms.md) | Check stored types, construction, reads, pointers, calls, and statements. |
| [Shared](shared.md) | Store or return pointers to immutable objects. |
| [ByValue](by-value.md) | Keep immutable value copies and use pointers for local reads. |
| [Owner](owner.md) | Publish a new ByValue object and read the current one. |
| [Frozen collections](frozen.md) | Build a slice or map, freeze it, and make a copy to edit. |
| [Unchecked immutable values](unchecked-values.md) | Hold a value that the analyzer trusts and does not prove. |

## Service forms (`supervise`)

- [Service results](service-results.md): result types for exported service methods.
- [Service references](service-references.md): stored references and direct calls to the current service.
- [Supervised services](services.md): execution and restart behavior.

## Rules for im and supervise

Use im type rules to check immutable data. Review service state changes and
synchronization in the service code.

Use the common forms with each type's forms. **Exhaustive** means all allowed
forms are listed. Unlisted operations on protected storage or service references
are rejected.

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
| Uses an im or supervise type, including inside another type | `var point api.Point`, where Point embeds `im.Shared` |
| Uses a function, interface, or generic type that mentions an im or supervise type | `callback := api.Publish`, where Publish takes `*api.Point`; `api.Alias[int]`, where Alias has an Owner type in its constraint |
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
- Protected values need a listed form: `reflect.ValueOf(1)` is allowed,
  but `reflect.ValueOf(p)` with a marked pointer fails the interface-boxing rule.

| Rejected examples — non-exhaustive | Use instead |
| --- | --- |
| Cgo in a package that imports or uses im | Ordinary typed Go code. Cgo produces one error at the source package line. |
| A function without a body whose signature contains an im type or a service reference | Supply a Go body to check its protected operations and service calls. |

An imported package's error includes one cause and a command to see its other errors.

Functions without a body use ordinary Go rules when their compiler-resolved
receiver, parameters, results, and generic constraints contain no im
type or service reference. For example, `func cpuID() uint64` and
`func checksum(data []byte) uint64` can use assembly implementations in a checked
package. A declaration such as `func read(value *Config) int`, where Config embeds
im.Shared, needs a Go body.
