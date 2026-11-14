# Language guide

Use immutago to keep object fields read-only and control where pointers can be
saved. Set the fields when you create an object. To change data, create another
object.

Choose the page for the value you need:

| Guide | Use it to |
| --- | --- |
| [Shared](shared.md) | Store or return pointers to immutable objects. |
| [Scoped](scoped.md) | Keep pointers local; pass them to named functions that do not save them. |
| [Owner](owner.md) | Publish a new Scoped object and read the current one. |
| [Frozen collections](frozen.md) | Build a slice or map, freeze it, and make a copy to edit. |
| [Supervised processes](processes.md) | Run mutable services with typed start arguments and immutable method messages. |
| [Common forms](common-forms.md) | Check stored types, construction, reads, pointers, calls, and statements. |

The common lists and each type's lists define the complete supported forms.
Other protected operations are rejected. Rejection examples are not exhaustive.
**Protected storage** means marked values, their fields, frozen collections,
and arrays containing them. Owner slots and holders have their own listed forms.

## Check the program

Run `immutago ./...` or `go vet -vettool=/absolute/path/to/immutago ./...`.
`go build` alone does not enforce these rules.

## Package boundary

A package is checked when any selection form below applies. This list is complete.
The fixed `im` and `proc` implementations, compiler-supplied standard library, and generated
Go test runners are trusted separately; user test packages still follow these rules.

| Selection form | Example |
| --- | --- |
| Direct `im` or `proc` import, including an alias or blank import | `import _ "github.com/lucasavila00/immutago/pkg/im"` |
| Used marked, frozen, Owner, or process type | `var point api.Point`, `var current api.Loader` |
| Used callback, interface, or generic contract containing an im or proc type | `callback := api.Publish`, `var value api.Alias[int]` when its constraint contains an Owner |
| Failed checked dependency, including through wrappers | Import a package whose immutable field proof failed. |

Aliases, generic arguments, and constraints remain part of the used type contract.
An ordinary scalar API does not select checking merely because its implementation
uses `im` or `proc` privately. CLI and vet use the same boundary.

Other packages are ordinary Go, including when selected by `immutago ./...`.
They may use `unsafe`, `reflect`, cgo, or functions without Go bodies. Their API
contracts are trusted, including string immutability and no hidden writes to
protected memory. The analyzer does not prove those implementations. Protected
values still follow their listed transport rules; ordinary libraries add no
suppression or trust-bypass annotations.

Checked packages reject `unsafe`, `reflect`, cgo, and functions without Go bodies.
Use checked typed Go code: replace `func readCount() int` with
`func readCount() int { return 1 }`, for example. A checked cgo package is rejected
as a whole.

A dependency diagnostic shows one representative original failure and a command
to check its origin package. Other dependency errors may exist.
