# Immutago

A Go immutability analyzer and frozen maps and slices. Embed `im.Marker` in a
struct. Build and change owned values with ordinary Go assignments; shared
references are read-only. Freeze exclusive native storage without copying.
Clone it when you need independent mutable storage.

The analyzer must run; `go build` alone does not enforce these rules.

[Get started](docs/getting-started.md) · [Language rules](docs/rules.md) ·
[Documentation](docs/README.md) · [Development](docs/development.md)
