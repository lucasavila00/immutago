# Example applications

These examples show complete application workflows. Shipping and search read
local files and serve HTTP requests. The market maker receives prices and fills
through a trading API; its tests drive the complete bot with a deterministic
fake. No live trading adapter is included.

| Application | Task |
| --- | --- |
| [Delivery quotes](shipping/README.md) | Price a shipment of several parcels, compare carrier services, and apply changes to zones and weight bands. |
| [Document search](search/README.md) | Search a directory of Markdown documents, rank results, and update the index when files change. |
| [HFT market maker bot](trading/README.md) | Quote prices, handle fills, share cash across instrument workers, and replace a strategy with its owned subscriptions and orders. |

Run a command from the repository root:

```sh
go run ./examples/shipping/cmd/shipping
```

```sh
go run ./examples/search/cmd/search
```

```sh
go test ./examples/trading -run TestBotQuotesFillsAndPolicy -count=1
```

Shipping listens on `127.0.0.1:8080`; search listens on `127.0.0.1:8081`.
Their READMEs give HTTP requests and changes to try. Press Ctrl+C to stop.
The market maker README explains its host API, trading contract, strategy
selection, and event-path cost tests.

For shipping and search, read the HTTP handler, then its calculation and loader.
Each request selects one complete immutable version. Each application uses one
supervised loader. For the market maker, read the trading API and Maker event
loop. Its independent policy service runs beside a tree of strategy and
instrument workers. Failed configuration edits keep the last valid settings.

Use the [language reference](../docs/reference/rules.md) for exact API forms. The application tests check
prices, search results, orders, fills, failed updates, concurrent reads, and shutdown. Checker
source forms and diagnostics belong in [the CLI fixtures](../tests/readme.md).

To regenerate records and check the examples:

```sh
go generate ./examples/...
go test -race ./examples/...
go run ./cmd/immutago ./examples/...
```

GitHub Actions runs the complete repository checks.
