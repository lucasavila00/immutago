# Example: HTTP prediction models

This program downloads a linear model, saves the validated JSON, and serves
predictions while it polls for updates. Failed HTTP or JSON reads leave the
current model available.

Run from the repository root. In one terminal, serve the sample file:

```sh
python3 -m http.server 8080 --directory examples/models/data
```

In another terminal, start the client:

```sh
go run ./examples/models/cmd/models -cache /tmp/immutago-model.json
```

The client uses input features `[2, 3]`. With the supplied coefficients `[2, 3]`,
it prints `score-v1: score 13`. Change the served JSON to
`{"name":"score-v2","weights":[4,1]}`. The next model produces score `11`.
Use two coefficients with this command. Press Ctrl-C to stop it. To use another
source, set `-url http://host/path/model.json`.

| File | Purpose |
| --- | --- |
| [fetcher.go](fetcher.go) | Fetch in Init, poll in Run, and publish each valid model through `im.Owner`. |
| [store.go](store.go) | Save the source document with a backend supplied by a factory. |
| [model/model.go](model/model.go) | Copy coefficients, wrap them with `im.Trust`, and expose read-only prediction. |
| [cmd/models/main.go](cmd/models/main.go) | Start the service with a logger, HTTP client, and store factory. |

The arguments hold a logger, HTTP client, and factory directly. The command uses
Go's logger and client, which permit concurrent use. Its factory captures only a
fixed path and creates a FileStore for that service. Use a separate output path
per service.

The model package promises with `im.Trust` that its copied
coefficients never change, and its API never exposes the slice. Only that
package can unwrap the pointer. The service publishes each model in a new
`snapshot` struct with `im.ByValue`. Each prediction reads one snapshot for the
whole calculation. The returned `Prediction` is a [service result](../../docs/service-results.md)
with its model name and score from that same version.

Check this project and run its tests:

```sh
go run ./cmd/immutago ./examples/models/...
go test -race ./examples/models/...
```

The tests use a real local HTTP server and check reloads, unchanged predictions
after HTTP, JSON, and store failures, saved documents, concurrent readers, a shared
recording store, shutdown during an HTTP request, and independence from the
caller's input slice. Repository CI runs these checks with the
other packages.
