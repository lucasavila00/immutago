# Runnable examples

These projects use the public im and supervise APIs in the repository's Go
module. Each has a command, a README, and tests for its real behavior.

| Project | Run and observe |
| --- | --- |
| [Configuration](configuration/README.md) | Edit JSON request limits while file refresh and quota windows run. |
| [Workers](workers/README.md) | Process text jobs, replace one pool's version, and remove a group while another keeps running. |
| [Models](models/README.md) | Fetch models over HTTP, save valid documents, and serve predictions during reloads. |

From the repository root:

```sh
go run ./cmd/immutago ./examples/...
go test -race ./examples/...
```

The existing CI jobs check these packages with the rest of the repository.
No separate module or service installation is needed. The model command uses
a local HTTP source; its README includes the start command.

## Guide examples

| Guide topic | Project source |
| --- | --- |
| File cache and background refresh | [FileCache](configuration/filecache.go), [command](configuration/cmd/limits/main.go) |
| Configuration loader and request limiter | [ConfigLoader](configuration/config.go), [Limiter](configuration/limiter.go) |
| Startup failure and reservation retry | [Startup and retry](configuration/README.md#startup-and-retry) |
| Child creation and queued work | [Pool](workers/pool.go), [Worker](workers/worker.go) |
| Changed configuration and version handover | [VariantConfig](workers/config.go), [Dispatcher](workers/dispatcher.go) |
| Concurrent dependencies, factories, and test backends | [Fetcher](models/fetcher.go), [Store](models/store.go), [tests](models/fetcher_test.go) |
| Package-owned trusted immutable data | [Model provider](models/model/model.go), [service consumer](models/fetcher.go) |

Analyzer acceptance/rejection fixtures stay in `tests/testdata`. They record
source forms and diagnostics; these projects show complete programs and test
their behavior.
