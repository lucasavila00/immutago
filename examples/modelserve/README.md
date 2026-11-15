# Local model serving

Serve house-price estimates from a primary model and an optional challenger. Each
request uses one complete coefficient set. The sample coefficients demonstrate
the application; they are not fitted from sales data. No external service is needed.

## Run and compare models

From the repository root:

```sh
go run ./examples/modelserve/cmd/modelserve
```

The command listens on `127.0.0.1:8082`. Use `-dir` to read a copy of the
[model files](models), `-addr` to change the address, or `-every` to change the
one-second poll interval.

In another terminal:

```sh
curl -s http://127.0.0.1:8082/predict \
  -H 'Content-Type: application/json' \
  -d '{"area_m2":90,"rooms":3}'
```

```json
{"version":"baseline-v1","estimate":247000}
```

Start the challenger arm while the primary keeps serving requests:

```sh
curl -s -X PUT http://127.0.0.1:8082/challenger \
  -H 'Content-Type: application/json' \
  -d '{"manifest":"challenger.json"}'
curl -s 'http://127.0.0.1:8082/predict?arm=challenger' \
  -H 'Content-Type: application/json' \
  -d '{"area_m2":90,"rooms":3}'
```

The challenger returns `{"version":"candidate-v1","estimate":252000}`.
`GET /status` lists active arms, their model versions, and any file errors.
A non-finite estimate returns HTTP 422 with an error instead of invalid JSON.

To replace the primary model, change its manifest from the repository root:

```sh
printf '{"model":"candidate.json"}\n' > examples/modelserve/models/primary.json
```

After the next poll, `/predict` returns `candidate-v1`. The switcher starts and
loads the new model before it changes the request route. It then stops the old
model's polling group. A request that already selected `baseline-v1` can finish
with those coefficients. Go keeps its immutable data alive; no reference counter
or request lease is needed.

Edits to the selected coefficient file are also loaded on each poll. All fields
are published together. A missing file or malformed JSON leaves the last valid
model available and sets the status error. A later valid file clears the error.
An invalid replacement manifest leaves the current model selected.

Remove only the challenger arm, then restore the sample primary manifest:

```sh
curl -i -X DELETE http://127.0.0.1:8082/challenger
printf '{"model":"baseline.json"}\n' > examples/modelserve/models/primary.json
```

The primary keeps serving. Press Ctrl+C in the first terminal to stop the server.
The command closes HTTP admission and waits for active requests before it stops
the supervised tree.

## Supervision and shared state

```text
root
├── primary group
│   └── Switcher: polls primary.json
│       └── selected version group
│           └── Model: polls the selected coefficient file
└── optional challenger group
    └── Switcher: polls challenger.json
        └── selected version group
            └── Model: polls the selected coefficient file
```

Groups exist for selective removal. They add no retry policy. A Model panic
restarts that Model; its switcher and peer arm keep running. A Switcher restart
stops its old model children before it initializes a replacement. The replacement
reads its manifest again and creates children under its new `Children()` group.
Each service uses the [runtime retry limit](../../docs/reference/services.md).
Exhaustion stops the root once and ends the process with the runtime's Failure.

The runtime stops groups and services in reverse start order. On application
shutdown, the challenger stops before the primary, and each model stops before
its switcher. The HTTP server is ordinary application code, outside this tree.

The router stores service references after Start returns. It needs no reserved
reference or self-reference. Router and switcher locks protect reference
publication. Model status reads its version and error under one lock. Prediction
requests keep a local `ParametersProjection`, so concurrent reloads do not change
their calculation. Review those locks and filesystem use as ordinary Go code;
immutable result types do not prove service isolation.

## Read and test the code

| Source | Responsibility |
| --- | --- |
| [http.go](http.go) | Predict, report status, and add or remove a challenger. |
| [router.go](router.go) | Select an arm and manage its lifetime group. |
| [switcher.go](switcher.go) | Read manifests, publish a new version, and retire the old group. |
| [model.go](model.go) | Read coefficients and calculate from one selected version. |
| [command](cmd/modelserve/main.go) | Open HTTP ingress and drain it before tree shutdown. |

The [lifecycle tests](lifecycle_test.go) check independent retries, parent cleanup,
selective stops, reverse shutdown order, and requests that keep an old model.
The [failure test](failure_test.go) uses a subprocess to check six failed attempts,
one terminal failure, and cleanup before exit. It uses the real runtime policy.
The [HTTP tests](http_test.go) check the workflow and errors. The
[command test](cmd/modelserve/main_test.go) holds an admitted HTTP request open
during cancellation and checks that it finishes before shutdown completes.

```sh
go generate ./examples/modelserve
go test -race ./examples/modelserve/...
go run ./cmd/immutago ./examples/modelserve/...
```

GitHub Actions runs the full repository checks.
