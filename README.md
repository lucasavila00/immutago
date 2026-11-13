# Immutago

A Go immutability analyzer and frozen maps and slices. Embed `im.Shared` for
immutable objects with shared references, or `im.LocalRef` when references
must not escape. Both are immutable from construction. Variables can be
reassigned; their objects cannot change. Freeze exclusive native storage without
copying. Clone it when you need a mutable builder.

The analyzer must run; `go build` alone does not enforce these rules.

[Get started](docs/getting-started.md) · [Language rules](docs/rules.md) ·
[Documentation](docs/README.md) · [Development](docs/development.md)
