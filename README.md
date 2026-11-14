# Immutago

Use immutago to make struct fields read-only and control where pointers can be
saved. Choose `im.Shared` to store or return pointers, or `im.Scoped` to keep
them local. Use `im.Owner[T]` to publish new Scoped objects. Frozen slices and
maps let you build data locally, then share it without copying the entries.

Go's garbage collector manages memory lifetime. Immutago adds rules for mutation
and pointer use; it does not guarantee prompt collection or prevent all leaks.

Run [supervised processes](docs/processes.md) with package `proc`; their exported
methods take and return only immutable values and PIDs. Give a process a logger
or a client with [`proc.Trust`](docs/processes.md#trusted-dependencies); the
checker does not prove that trust.
The checker does not check package variables, so processes can still share a
mutable global. See [what is not checked](docs/processes.md#not-checked).

The analyzer must run; `go build` alone does not enforce these rules.

[Get started](docs/getting-started.md) · [Language guide](docs/rules.md) ·
[Documentation](docs/README.md) · [Development](docs/development.md)
