# Language guide

[Documentation](README.md) · [Get started](getting-started.md) · [API](../pkg/im/readme.md)

Immutago restricts mutation and reference retention beyond Go's memory safety.
Objects are immutable from construction; whole variable bindings can change.
Go's garbage collector manages object lifetime.

Choose the page for the value you need:

| Guide | Use it to |
| --- | --- |
| [Shared](shared.md) | Store or return pointers to immutable objects. |
| [Scoped](scoped.md) | Keep immutable object pointers local or in direct non-retaining calls. |
| [Owner](owner.md) | Publish fresh Scoped objects through an atomic slot. |
| [Frozen collections](frozen.md) | Freeze an exclusive native builder, read it, or Clone mutable storage. |
| [Common forms](common-forms.md) | Check stored types, construction, reads, pointers, calls, and statements. |

The common lists and each type's lists define the complete supported forms.
Other protected operations are rejected. Rejection examples are not exhaustive.
**Protected storage** means marked values, their fields, frozen collections,
and arrays containing them. Owner slots and holders have their own listed forms.

## Check the program

Run `immutago ./...` or `go vet -vettool=/absolute/path/to/immutago ./...`.
`go build` alone does not enforce these rules. There are no suppression or trust
annotations. Unsupported forms fail rather than being skipped.

Checked code cannot import `unsafe` or `reflect`, or declare functions without
Go bodies—even for scalar-only functions. Replace `func readCount() int` with
a checked Go implementation such as `func readCount() int { return 1 }`.

Cgo packages are rejected as a whole, with one error at a source `package`
line. Replace `import "C"` and its C calls with checked Go code.

A dependency diagnostic shows one representative original failure, its source
location, and a command to check the original package. Other dependency errors
may exist. Reports from direct package analysis do not change.
