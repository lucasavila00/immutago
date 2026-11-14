# Live configuration and request limits

Three services turn a JSON file into a request quota:

| Service | Work |
| --- | --- |
| [FileCache](filecache.go) | Read the file during Init; refresh its text in Run. |
| [ConfigLoader](config.go) | Parse changed text and publish immutable limits through `im.Owner`. Keep the last valid limits if an edit is invalid. |
| [Limiter](limiter.go) | Count requests under a mutex; reset the count once per time window. Read the current limit on each request. |

The services use `supervise.Ref` to call each other. Calls run in the caller's
goroutine. Each service protects its own mutable fields.

## Run

From the repository root:

```sh
go run ./examples/configuration/cmd/limits
```

The command reads [limits.json](limits.json), makes five requests per second,
and prints the decision and counters. The `api` limit starts at three requests
per one-second window. Change it to `5` while the command runs; later requests
use the new limit. Press Ctrl+C to stop all services.

To use a different file or key:

```sh
go run ./examples/configuration/cmd/limits -config /path/to/limits.json -key batch
```

The JSON object maps keys to nonnegative integer limits; `null` is rejected. A missing key has a
zero limit. A lower limit does not erase requests already counted in the current
window. A file read failure makes FileCache restart; invalid JSON keeps the last
valid configuration and reports the parse error.

## Startup and retry

FileCache reads the file in Init. A missing file returns an error before callers
can read the reference. A failed Init leaves a reserved reference available:

```go
cache := supervise.Reserve[configuration.FileCache]()
args := configuration.FileArgs{Path: path, Every: time.Second}
err := supervise.StartReserved(root, cache, args)
if errors.Is(err, os.ErrNotExist) {
    if err := os.WriteFile(path, []byte(`{"api":3}`), 0600); err != nil {
        panic(err)
    }
    err = supervise.StartReserved(root, cache, args)
}
if err != nil { panic(err) }
```

Retry only after failure; starting the same successful reservation twice panics.
The startup test creates a missing file and retries the same reservation, then
reads its initial contents.

## Check

```sh
go run ./cmd/immutago ./examples/configuration/...
go test -race ./examples/configuration/...
```

The [tests](configuration_test.go) check actual file changes, invalid edits,
window resets, concurrent quota use, startup errors, and reservation retry.
Repository CI runs these commands' checks as part of its existing jobs.

JSON decoding uses an ordinary Go map. ConfigLoader copies its scalar entries
into a fresh builder before Freeze, so no decoder alias can change a published
configuration. Each successful reload costs one map copy; reads use the frozen
map directly.
