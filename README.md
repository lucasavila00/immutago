# Immutago

A Go analyzer that restricts mutation and reference retention, with frozen maps
and slices. `im.Shared` permits stored and returned pointers. `im.Scoped`
restricts pointers to local use and direct non-retaining calls; `im.Owner[T]`
publishes fresh Scoped objects through an atomic slot. Both marker modes make
objects immutable from construction. Variables can be reassigned.

These restrictions add to Go's memory safety. Go's garbage collector manages
object lifetime; immutago does not guarantee prompt collection or prevent all
memory leaks. Freeze exclusive native storage without copying. Clone it when
you need a mutable builder.

Run [supervised processes](docs/processes.md) with package `proc`; their exported
methods take and return only immutable values and PIDs.

The analyzer must run; `go build` alone does not enforce these rules.

[Get started](docs/getting-started.md) · [Language guide](docs/rules.md) ·
[Documentation](docs/README.md) · [Development](docs/development.md)
