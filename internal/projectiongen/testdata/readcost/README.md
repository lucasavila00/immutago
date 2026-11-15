# Generated read cost

This consumer compares generated field reads with native atomic pointer reads.
The roots have the same field layout. The two root sizes detect unwanted copies
of the root or its array. Store runs before the timer starts.

Run from this directory:

```sh
go generate .
go test -run '^$' -bench . -benchmem
```

Run the checker from the repository root:

```sh
go run ./cmd/immutago ./internal/projectiongen/testdata/readcost
```

Record the Go version and platform with measurements. This package is under
testdata, so `go test ./...` does not run these benchmarks.
