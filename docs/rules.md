# Language reference

| Guide | Use it to |
| --- | --- |
| [Shared](shared.md) | Store or return pointers to immutable objects. |
| [Scoped](scoped.md) | Keep pointers local; use functions and methods that do not save them. |
| [Owner](owner.md) | Publish a new Scoped object and read the current one. |
| [Frozen collections](frozen.md) | Build a slice or map, freeze it, and make a copy to edit. |
| [Supervised services](services.md) | Learn the execution and restart model. |
| [Service forms](service-forms.md) | Look up supported service declarations and calls. |
| [Unchecked immutable values](unchecked-values.md) | Hold a value that the analyzer trusts and does not prove. |
| [Common forms](common-forms.md) | Check stored types, construction, reads, pointers, calls, and statements. |

The common lists and each type's lists define the complete supported forms.
Other protected operations are rejected. Rejection examples are not exhaustive.

| Term | Meaning |
| --- | --- |
| Marked struct | A struct that embeds `im.Shared` or `im.Scoped`. |
| Protected storage | Marked values, their fields, frozen collections, and arrays containing them. |
| Service dependency | A `supervise.ConcurrentSafe[T]` value. Its uses are checked; its contents are trusted. |
| Owner slot and holder | An `im.Owner[T]` field or variable, and a struct that contains one. See [Owner](owner.md). |

## Trust markers

User code has exactly two trust markers. The checker does not prove either
promise:

| Marker | Promise | Where |
| --- | --- | --- |
| [`im.UncheckedValue[T]`](unchecked-values.md), from `im.Trust` | The value never changes after `im.Trust`. Only the package that declares `T` can wrap or read it. | Anywhere a stored type is allowed. |
| [`supervise.ConcurrentSafe[T]`](services.md#concurrent-dependencies), from `supervise.AssumeConcurrentSafe` | The value is safe for concurrent use by several services. | Only in service start arguments. |

## Check the program

Run `immutago ./...` or `go vet -vettool=/absolute/path/to/immutago ./...`.
`go build` alone does not enforce these rules.

## Package boundary

The checker checks a package in any of the cases below. This list is complete.
It also checks your test files.

| Check the package when it… | Example |
| --- | --- |
| Imports `im` or `supervise`, including an alias or blank import | `import _ "github.com/lucasavila00/immutago/pkg/im"` |
| Uses an im or supervise type, including inside another type | `var point api.Point`, where Point embeds `im.Shared` |
| Uses a function, interface, or generic type that mentions an im or supervise type | `callback := api.Publish`, where Publish takes `*api.Point`; `api.Alias[int]`, where Alias has an Owner type in its constraint |
| Imports a package that failed its checks, directly or through another package | Import a package with an invalid immutable field. |

- Aliases, type arguments, and generic constraints also count.
- The CLI and `go vet` use the same rules.
- Other packages are trusted to obey Go contracts, including immutable strings.
- The `im`, `supervise`, and standard-library implementations are trusted.
- The generated Go test runner is skipped; user tests are checked.
- User code has no suppression comments. Its only trust markers are
  `im.UncheckedValue` and `supervise.ConcurrentSafe`.

Checked packages reject these forms; this rejection list is not exhaustive:

| Rejected form | Use instead |
| --- | --- |
| `unsafe`, `reflect`, or cgo | Ordinary typed Go code. Cgo produces one error at the source package line. |
| A function without a body whose signature contains an im or supervise type | Supply a Go body to check its protected operations. |

An imported package's error includes one cause and a command to see its other errors.

Functions without a body use ordinary Go rules when their compiler-resolved
receiver, parameters, results, and generic constraints contain no im or supervise
type. For example, `func cpuID() uint64` and `func checksum(data []byte) uint64`
can use assembly implementations in a checked package. A declaration such as
`func read(value *Config) int`, where Config embeds im.Shared, needs a Go body.
