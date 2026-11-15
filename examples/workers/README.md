# Versioned text workers

This program runs text jobs in child worker pools. Version 1 converts text to
uppercase; version 2 converts it to lowercase. The dispatcher can replace one
variant's pool or remove it while other variants continue to serve jobs.

From the repository root:

```sh
go run ./examples/workers/cmd/workers
go test -race ./examples/workers/...
go run ./cmd/immutago ./examples/workers/...
```

The command prints:

```text
blue v1: HELLO WORLD
blue v2: hello world
blue removed: true
green v1: STILL RUNNING
```

| Source | Task |
| --- | --- |
| [Worker](worker.go) | Receive jobs on a Go channel and return converted text. |
| [Pool](pool.go) | Create child workers during Init and distribute requests among them. |
| [Configuration](config.go) | Publish immutable variant-name snapshots through `im.Owner`. |
| [Dispatcher](dispatcher.go) | Poll configuration, stage new versions, and stop old or removed groups. |
| [Command](cmd/workers/main.go) | Start two variants, replace blue, remove blue, and read green. |
| [Tests](workers_test.go) | Check job results, shutdown, readiness, failed setup, and concurrent changes. |

`Pool.Init` creates the workers under `p.Children()`. Each worker starts its job
loop in Run. Pool readiness uses non-blocking checks, so a stop between Init and
Run cannot leave a caller waiting for a child that will never start.

`Dispatcher.Load(name, version)` stages a new group. It does not switch the
serving version. `Refresh`, called by the dispatcher's Run loop, publishes the
candidate only after all its workers are ready. It then stops the old group.
Failed setup leaves the serving version unchanged. Repeated requests for the
current or pending version do nothing.

The dispatcher serializes configuration changes and version requests. Request
calls use a separate mutex; they do not wait while another group stops. A request
already sent to an old worker can finish or return `Available: false` during
shutdown. Callers can retry against the dispatcher's current pool.

`VariantConfig` outlives the dispatcher. A restarted dispatcher reads its current
names and creates fresh version-1 pools; version requests are not persisted.
Removing a name stops both its serving pool and any pending replacement. Start
arguments use ordinary Go structs; job channels stay inside each worker.
Exported methods return one Scoped result: a scalar, a frozen collection, or the
immutable Result struct.
