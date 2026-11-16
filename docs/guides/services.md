# Run a background service

The [shipping loader](../../examples/shipping/loader.go) reads a catalog before
HTTP requests begin, then checks for edits in the background. `supervise` manages
that background work and its restarts. This guide follows the loader's lifecycle;
the [application README](../../examples/shipping/README.md) gives the run commands.

## Set up and run

The loader embeds `supervise.Service[Args]` and declares `Init` and `Run`:

```go
type Loader struct {
    supervise.Service[Args]
    current CatalogCell
}
```

`Init` reads and validates the first catalog. An error prevents startup. `Run`
uses a ticker to try later reloads and returns when its context ends. Expected
file errors are logged; the last valid catalog stays available. A panic in Run
causes a restart with a new Loader value and the same arguments.

The [command](../../examples/shipping/cmd/shipping/main.go) starts the loader
under a root group and waits for shutdown when it returns:

```go
root := supervise.NewRoot(nil)
defer func() { <-root.Stop() }()
loader, err := supervise.Start[shipping.Loader](root, shipping.Args{Path: path, Every: time.Second})
if err != nil {
    return err
}
```

Start calls Init before it returns the reference. The command creates the HTTP
server only after Start succeeds. On shutdown, it closes the HTTP server before
it stops the root group. `<-root.Stop()` waits for background work; it does not
wait for HTTP handlers or other direct method calls.

## Keep a reference and call a method

The handler keeps `supervise.Ref[Loader]`. Each request calls:

```go
catalog := loader.Borrow().Catalog()
```

Borrow selects the current service after any restart. Catalog runs in the
request's goroutine and loads a projection from the Cell. That projection fixes
the catalog version for the quote. Keep the reference for later requests and
select again. The [reference rules](../reference/service-references.md) cover
optional references and direct Borrow calls.

## Review shared state and inputs

Service methods can run at the same time as Run and other callers. Immutable
results do not make mutable service fields safe. Review field access, locks,
saved inputs, and dependency calls in the service implementation.

The [search indexer](../../examples/search/indexer.go) shows this distinction.
Its Cell publishes an immutable index. A mutex also protects the error text and
keeps Status consistent with publication. Its `fs.FS` start argument follows the
filesystem's ordinary Go contract; no immutable wrapper is required.

Exported service methods can return any number of read results and errors, in any order.
The result rule does not restrict ordinary method inputs, callbacks, or dependency
calls. See [service results](../reference/service-results.md) for the full list.

## Choose a group lifetime

The example has one service under the root. Use `supervise.StartGroup(parent)`
when several services must stop together. Inside a service, use `s.Children()`
for work that must end with that run. A restart creates a new service value and
child group, so create those children again. Other services keep running.
Groups control shutdown only; they do not restart.

The [runtime reference](../reference/services.md) defines child shutdown order,
restart limits and failure reports. In tests,
[`supervisetest.NewRoot(t)`](../../pkg/supervise/supervisetest/readme.md) stops the
tree during cleanup. It fails the test and prints the original stack when a
supervised attempt fails. It also fails if shutdown takes more than five seconds.
Tests that expect supervised failures can use `supervise.NewRoot` with their own reporter.
