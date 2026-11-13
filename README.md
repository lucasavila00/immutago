# Immutago

A Go immutability analyzer and frozen maps and slices. Embed `im.Marker` in a
struct. Construct immutable objects, then share references to them. Variables
can be reassigned; the objects they reference cannot change. Freeze exclusive
native storage without copying. Clone it when you need a mutable builder.

The analyzer must run; `go build` alone does not enforce these rules.

[Get started](docs/getting-started.md) · [Language rules](docs/rules.md) ·
[Documentation](docs/README.md) · [Development](docs/development.md)
