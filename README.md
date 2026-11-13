# Immutago

A Go immutability analyzer and immutable collections. Embed
`im.Marker` in a struct; the analyzer checks field types and borrowing. Build owned values with ordinary
Go assignments; shared references are read-only. No escape hatches.

```sh
go run ./cmd/immutago ./...
make test-e2e
```

See the [contract](docs/guide/immutability.md),
[development guide](docs/contrib/development.md), and
[working rules](AGENTS.md).
