# Local model serving

This example will serve house-price estimates from a primary model and an optional
challenger. It will use local JSON files and an ordinary HTTP server.

The supervisor owns this tree:

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

The switcher starts a replacement model, publishes its reference, and then stops
the old version group. Each request selects one immutable coefficient set.
Requests that selected the old version can finish after its background work stops.
A parent restart stops its old children before creating replacements. Each service
uses the runtime's own retry limit. Groups add no retry policy.

Implementation is in progress in the draft PR. The service and router code is an
initial checkpoint. The remaining work is:

- Add the HTTP command, prediction and status routes, and challenger controls.
- Add commands that exercise model reload, replacement, and selective shutdown.
- Test prediction results, file errors, restarts, child cleanup, retry exhaustion,
  and in-flight requests during replacement.
- Simplify numeric records in the existing shipping and search examples with
  `im.Copied`, then check that extracted copies can change independently.
- Complete hosted validation and independent review.
