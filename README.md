# Immutago

A Go immutability analyzer and immutable collections. Embed
`immutable.Marker` in a struct; the analyzer checks its fields and rejects
mutations and unsupported aliases. No escape hatches.

```sh
go run ./cmd/immutago ./...
make test-e2e
```

See the [contract](docs/guide/immutability.md),
[development guide](docs/contrib/development.md), and
[working rules](AGENTS.md).
