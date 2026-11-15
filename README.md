# Immutago

A Go checker and library for immutable data and supervised services.

The design goal is local review: check immutable data through its type rules;
review service state changes and locks in the service code.

| What do you need? | Start here |
| --- | --- |
| Frozen records, strings, slices, and maps | [Immutable data with im](docs/README.md#immutable-data-im) |
| Background services, restarts, and child groups | [Supervised services with supervise](docs/README.md#supervised-services-supervise) |

[Install the checker](docs/getting-started.md#install), then run `immutago ./...`.
`go build` does not enforce these rules.

[Example projects](examples/README.md) use both packages in tested programs.
[Glossary](docs/glossary.md) defines the terms used in the guides and diagnostics.
[Maintainer documentation](docs/README.md#maintainers) covers development and review.
