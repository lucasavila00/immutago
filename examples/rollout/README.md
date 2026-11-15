# Checkout release monitor

An online store wants to replace `checkout-v1` with `checkout-v2`, a new version
of the same checkout application. Before the operator sends more traffic to v2,
they want to check its server errors and response times against v1.

In this workflow, both versions already run and receive requests. The operator
controls application traffic through a separate router. A **replica** is one
running instance of a version. Each replica writes an **access log** with the
HTTP status and response time for each request.

This example is the monitor for that workflow. It reads the logs, combines the
results for each version, and serves the comparison at `GET /report`.

| Role in `rollout.json` | Meaning |
| --- | --- |
| `primary`, initially `checkout-v1` | The existing version used as the comparison baseline. |
| `candidate`, here `checkout-v2` | The new version under evaluation. |

The **gate** checks the comparison against the rules in `policy.json`. These
rules require enough requests from both versions and limit increases in errors
and response time. A `pass` result means the candidate meets those rules for the
recorded requests. The operator uses the result to decide whether to send more
traffic to it. The monitor does not route requests or enforce that decision.

The local example uses sample logs in place of running checkout servers.
You append log rows to simulate new requests. A background policy service
updates the limits, while a separate [service tree](#service-ownership) reads
the logs and updates the report.

## Run and compare

From the repository root, copy the [sample data](data) into a temporary directory:

```sh
demo=$(mktemp -d)
cp examples/rollout/data/* "$demo/"
printf 'Data directory: %s\n' "$demo"
go run ./examples/rollout/cmd/rollout -dir "$demo"
```

In another terminal, set `demo` to the printed directory, then read the report:

```sh
demo=/path/printed/above
curl -s http://127.0.0.1:8082/report
```

The primary is `checkout-v1`, with two replicas, 10 requests, no failures, and
`p95_ms: 100`. The verdict is `no_candidate`. Add the candidate's logs to the
comparison:

```sh
cat > "$demo/rollout.json" <<'JSON'
{
  "primary": {"version": "checkout-v1", "logs": ["v1-a.ndjson", "v1-b.ndjson"]},
  "candidate": {"version": "checkout-v2", "logs": ["v2-a.ndjson", "v2-b.ndjson"]}
}
JSON
```

After one second, the report shows six candidate requests and `verdict: waiting`.
The policy requires at least 10 requests from each release. Simulate four more
candidate requests by appending rows to its log:

```sh
cat >> "$demo/v2-a.ndjson" <<'JSON'
{"status":200,"duration_ms":30}
{"status":200,"duration_ms":35}
{"status":200,"duration_ms":40}
{"status":200,"duration_ms":45}
JSON
curl -s http://127.0.0.1:8082/report
```

After the next read, the candidate has 10 requests, no failures, and `p95_ms: 50`.
The verdict becomes `pass`. Each response contains one complete selection and
decision, including `policy_version` and any configuration errors.

## Change policy and promote

The simple policy service runs beside the complete deployment tree. Change its
file to require more data:

```sh
cat > "$demo/policy.json" <<'JSON'
{
  "version": "larger-sample",
  "min_requests": 20,
  "max_error_increase_percent": 1,
  "max_p95_increase_ms": 50
}
JSON
```

Within two read intervals, the verdict becomes `waiting`. No deployment or
reader restarts. Restore `min_requests` to 10, then make the candidate the
monitor's new comparison baseline:

```sh
cat > "$demo/rollout.json" <<'JSON'
{
  "primary": {"version": "checkout-v2", "logs": ["v2-a.ndjson", "v2-b.ndjson"]}
}
JSON
```

The current candidate becomes primary with its existing readers and counts.
Rollout publishes that selection before it stops the old primary and closes
both old log streams. A request that selected the old report can still finish.
Promotion is an operator action; the monitor does not prevent a selection when
the verdict is `fail` or `waiting`.

To stop monitoring the candidate instead, remove `candidate` from the file and
keep the original `primary`. Only the candidate's monitoring services stop. You
can add another candidate later without restarting the monitor.

Use `-addr` to change the listen address or `-every` to change the one-second read
interval. Without `-dir`, the command uses the repository's sample data directly.
Press Ctrl+C to close HTTP admission and give active requests up to five seconds
to finish. The command then stops the service tree. An unexpected HTTP listener
error uses the same drain path.

## Data and decisions

Each selected log is an append-only file with one JSON object per line. A record
needs `status` from 100 through 599 and a nonnegative integer `duration_ms`.
Extra JSON fields are ignored. A final row without a newline waits for the rest
of the row. Counts include all complete records in the file, with no time window.

The gate uses these rules, in this order:

1. An invalid row in either release gives `invalid_logs`. Invalid rows do not
   count as requests.
2. Fewer than `min_requests` valid requests in either release gives `waiting`.
3. Candidate error percentage minus primary error percentage must be no greater
   than `max_error_increase_percent`. This limit uses percentage points.
4. Candidate p95 minus primary p95 must be no greater than `max_p95_increase_ms`.
   Exceeding either comparison limit gives `fail`; otherwise the result is `pass`.

Latency buckets end at 10, 25, 50, 100, 250, 500, and 1000 ms. The p95 value is
the upper bound of the bucket containing the 95th percentile. The last bucket
uses the largest observed duration. This is a conservative histogram result,
not an exact percentile of retained request records.

Keep selected logs append-only. To correct old records, write a corrected file
at a new path and change that deployment's `logs` list, or stop and restart the
application after the repair. Editing, truncating, or replacing a selected file
does not reset its open stream. The example has no log rotation mechanism.

A bad `rollout.json` edit or an unreadable new deployment keeps the current
selection and sets `problem`. A bad policy edit keeps the last valid policy and
sets `policy_problem`. A valid edit clears the corresponding error. Invalid
initial configuration prevents startup. A reader I/O failure closes its stream
and causes a supervisor retry. Replay replaces that replica's cumulative totals;
it does not add them again. Reports can show the last published counts while a
reader waits to retry.

## Service ownership

This tree runs the monitor's background work. Its Deployment and Reader services
process log files from the checkout replicas. Policy reads the gate rules
independently, so the operator can change the limits while log reading continues.

```text
root
├── Policy: refresh policy.json and publish immutable limits
└── Rollout: apply release selection and evaluate the live gate
    ├── primary lifetime group
    │   └── Deployment: combine this release's replica totals
    │       ├── Reader: own and read replica A's log stream
    │       └── Reader: own and read replica B's log stream
    └── optional candidate lifetime group
        └── Deployment
            ├── Reader
            └── Reader
```

Each service has a separate job. Readers own open streams and offsets.
Deployments combine their current reader results after update signals. Rollout
owns release changes and the comparison; HTTP code only reads reports. Policy
is the simple independent root branch. It starts first, so it stops after its
consumer tree.

One reader retry leaves its peers running. A Deployment retry first stops all
readers from the old attempt, then reconstructs its complete sum. A Rollout retry
stops both owned deployment trees before it recreates them; Policy continues.
The [runtime](../../docs/reference/services.md) supplies all retry policy.
Groups only control lifetimes. Six failures within the runtime window stop the
root once, close its resources, and terminate the process for an external restart.

`Totals` uses `im.Copied` for independent numeric results. `PolicySnapshot` and
`Report` use Frozen records and generated Cells. A report contains data, so an
old projection survives service retirement. Ordinary service fields, filesystem
dependencies, and the update channels still need normal Go review. The checker
does not prove service isolation.

## Read and check the code

| Source | Responsibility |
| --- | --- |
| [policy.go](policy.go) | The simple independent configuration service. |
| [rollout.go](rollout.go), [config.go](config.go) | Selection, dynamic child ownership, and publication before retirement. |
| [deployment.go](deployment.go) | Combine complete replica snapshots after updates. |
| [reader.go](reader.go), [metrics.go](metrics.go) | Open streams, incremental records, replay, counts, and histogram math. |
| [report.go](report.go), [http.go](http.go) | Gate rules and one immutable selection per HTTP response. |
| [command](cmd/rollout/main.go) | Start both branches and drain ingress before shutdown. |

[Workflow tests](workflow_test.go) check data and policy changes.
[Lifecycle tests](lifecycle_test.go) check live work during independent retries,
cleanup before replacement, selective removal, promotion, and old request data.
The [failure test](failure_test.go) checks root exhaustion in a subprocess; the
[command tests](cmd/rollout/main_test.go) check both HTTP drain paths.

```sh
go generate ./examples/rollout
go run ./cmd/immutago ./examples/rollout/...
go test ./examples/rollout/...
```

GitHub Actions runs the full repository and race checks.
