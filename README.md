# Immutago

A Go checker and library for immutable data and supervised services.

| Task | Guide |
| --- | --- |
| Install the checker and run a complete example | [Get started](docs/getting-started.md) |
| Share read-only objects | [Shared](docs/shared.md) |
| Prevent functions from saving object pointers | [Scoped](docs/scoped.md) |
| Replace data while readers finish with the previous version | [Owner](docs/owner.md) |
| Build a slice or map, then make it read-only | [Frozen collections](docs/frozen.md) |
| Run a service with supervised restarts | [Processes](docs/processes.md) |
| Give a process a logger or a client that the checker does not check | [Unchecked dependencies](docs/processes.md#unchecked-dependencies) |

Run `immutago ./...` to check your code. `go build` does not enforce these rules.

See the [documentation index](docs/README.md) for the complete language reference,
API reference, and maintainer docs.
