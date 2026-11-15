# Example applications

These applications read local files and serve HTTP requests. Each includes sample
data, a command, request and response examples, and tests for its application
behavior. They run in the repository's Go module without an external service.

| Application | Task |
| --- | --- |
| [Delivery quotes](shipping/README.md) | Price a shipment of several parcels, compare carrier services, and apply changes to zones and weight bands. |
| [Document search](search/README.md) | Search a directory of Markdown documents, rank results, and update the index when files change. |
| [Live release gate](rollout/README.md) | Compare release health from replica access logs, change policy, and add or retire deployment trees. |

Run a command from the repository root:

```sh
go run ./examples/shipping/cmd/shipping
```

```sh
go run ./examples/search/cmd/search
```

```sh
go run ./examples/rollout/cmd/rollout
```

Shipping listens on `127.0.0.1:8080`, search on `127.0.0.1:8081`, and the
release gate on `127.0.0.1:8082`.
Each README gives a request to send from another terminal and a change to try
while the application runs. Press Ctrl+C to stop.

Read an application's HTTP handler first, then follow its calculation and loader.
Each request selects one complete immutable version. Shipping and search each
use one supervised loader. The release gate runs a simple policy service beside
a tree of deployments and log readers. Failed configuration updates leave the
last valid data available to requests.

Use the [language reference](../docs/reference/rules.md) for exact API forms. The application tests check
prices, search results, release decisions, failed updates, concurrent reads, and shutdown. Checker
source forms and diagnostics belong in [the CLI fixtures](../tests/readme.md).

To regenerate records and check the examples:

```sh
go generate ./examples/...
go test -race ./examples/...
go run ./cmd/immutago ./examples/...
```

GitHub Actions runs the complete repository checks.
