# Immutago

A Go checker and library for immutable data and supervised services.

The design goal is local review: check immutable data through its type rules;
review service state changes and locks in the service code.

| What do you need? | Read |
| --- | --- |
| Install the checker and run a first program | [Get started](docs/guides/getting-started.md) |
| Learn the APIs | [Guides](docs/README.md#guides) |
| Look up exact rules and signatures | [Reference](docs/README.md#reference) |
| Run a complete application | [Delivery quotes and document search](examples/README.md) |
| Change the checker or runtime | [Maintainer documentation](docs/README.md#maintainers) |

[Install the checker](docs/guides/getting-started.md#install), then run `immutago ./...`.
`go build` does not enforce these rules.

[Glossary](docs/reference/glossary.md) defines the terms used in the documentation
and diagnostics.
