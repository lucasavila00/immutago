# Language reference

| Guide | Use it to |
| --- | --- |
| [Shared](shared.md) | Store or return pointers to immutable objects. |
| [Scoped](scoped.md) | Keep pointers local; pass them to named functions that do not save them. |
| [Owner](owner.md) | Publish a new Scoped object and read the current one. |
| [Frozen collections](frozen.md) | Build a slice or map, freeze it, and make a copy to edit. |
| [Supervised processes](processes.md) | Learn the execution and restart model. |
| [Process forms](process-forms.md) | Look up supported process declarations and calls. |
| [Opaque values](opaque.md) | Hold a value that the analyzer trusts and does not prove. |
| [Common forms](common-forms.md) | Check stored types, construction, reads, pointers, calls, and statements. |

The common lists and each type's lists define the complete supported forms.
Other protected operations are rejected. Rejection examples are not exhaustive.
**Protected storage** means marked values, their fields, frozen collections,
and arrays containing them. Owner slots and holders have their own listed forms.

## Trust markers

User code has exactly two trust markers. The checker does not prove either
promise:

| Marker | Promise | Where |
| --- | --- | --- |
| [`im.Opaque[T]`](opaque.md), from `im.Seal` | The value never changes after `im.Seal`. Only the package that declares `T` can seal or read it. | Anywhere a stored type is allowed. |
| [`proc.UncheckedDependency[T]`](processes.md#unchecked-dependencies), from `proc.Trust` | The value is safe for concurrent use by several processes. | Only in process start arguments. |

## Check the program

Run `immutago ./...` or `go vet -vettool=/absolute/path/to/immutago ./...`.
`go build` alone does not enforce these rules.

## Package boundary

The checker checks a package in any of the cases below. This list is complete.
It also checks your test files.

| Check the package when it… | Example |
| --- | --- |
| Imports `im` or `proc`, including an alias or blank import | `import _ "github.com/lucasavila00/immutago/pkg/im"` |
| Uses an im or proc type, including inside another type | `var point api.Point`, where Point embeds `im.Shared` |
| Uses a function, interface, or generic type that mentions an im or proc type | `callback := api.Publish`, where Publish takes `*api.Point`; `api.Alias[int]`, where Alias has an Owner type in its constraint |
| Imports a package that failed its checks, directly or through another package | Import a package with an invalid immutable field. |

Aliases, type arguments, and generic constraints also count. The CLI and
`go vet` use the same rules. Other packages are trusted to follow Go rules,
including not changing string bytes through `unsafe`.

The checker trusts the implementations of `im`, `proc`, and the Go standard
library. It also skips the test runner that Go generates. There is no comment
or annotation that lets user code bypass checks. The only trusted values in user
code are the two [trust markers](#trust-markers).

In a checked package, `unsafe`, `reflect`, cgo, and functions without Go bodies
are rejected. For example, write `func readCount() int { return 1 }` instead of
`func readCount() int`. A cgo package gets one package-level error.

If an imported package fails, the error shows one cause and a command to check
that package. Run that command to see its other errors.
