# Get started

## Install

In a Go module, add the library and generator, then install the checker:

```sh
go get -tool github.com/lucasavila00/immutago/cmd/immutago-gen@latest
go install github.com/lucasavila00/immutago/cmd/immutago@latest
```

## Example: typed user reads

This program publishes immutable user data and reads it through generated
methods. The checker rejects field writes and requires current generated code.

Save this as `main.go`:

```go
package main

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import "github.com/lucasavila00/immutago/pkg/im"

type User struct {
    im.Frozen
    Name  im.FrozenString
    Score int
}

var current UserCell

func main() {
    current.Store(User{Name: im.FreezeString("alice"), Score: 10})
    user := current.Load()
    println(user.Name().Unsafe(), user.Score())
}
```

Generate `zz_immutago.gen.go`, then check and run the program:

```sh
go generate ./...
immutago ./...
go run .
```

The program prints `alice 10`.

- `im.Frozen` marks immutable stored fields.
- The generator supplies UserCell and UserProjection. Load returns a local UserProjection.
- Name returns StringProjection; Unsafe supplies its shared native string to println.
- Score returns a copied integer. FreezeString copies no bytes.

Run generation again after a record changes. Keep the generated file with the
source. Missing or stale output fails the checker. After the last Frozen declaration
is removed, remove the directive and generated file together, or keep the directive
and generate once more. See [required generation](../reference/rules.md#required-generation).

Continue with [Publish immutable data](immutable-data.md) to follow construction,
publication, and reads in the shipping application.

## Check in CI

```sh
immutago ./...
```

- Errors make the command fail.
- Run it for each platform and set of build tags you use.
- Generate with the same build configuration first. Use separate checkouts if
  configurations need different generated output.
- `go build` does not run the checker.

Or use `go vet`:

```sh
go vet -vettool=/absolute/path/to/immutago ./...
```

For build tags, set `GOFLAGS` for the CLI or pass `-tags` to `go vet`:

```sh
GOFLAGS=-tags=integration go generate ./...
GOFLAGS=-tags=integration immutago ./...
go vet -vettool=/absolute/path/to/immutago -tags=integration ./...
```

`immutago -tags integration ./...` does not select tagged files. Use one of the
commands above to check them, including packages whose files all require a tag.
One generated file can serve all configurations when the generated declarations
and their type dependencies stay the same. See
[required generation](../reference/rules.md#required-generation) for configurations
with different schemas and packages whose declarations all require a tag.

Run generation and checks on each target platform in CI. For a cross build of one
package, first build the tools for the host, then select the target:

```sh
go build -o ./bin/immutago-gen github.com/lucasavila00/immutago/cmd/immutago-gen
go build -o ./bin/immutago github.com/lucasavila00/immutago/cmd/immutago
GOOS=windows GOARCH=amd64 ./bin/immutago-gen
GOOS=windows GOARCH=amd64 ./bin/immutago .
```

Do not use the `go run` generation directive to execute a cross-built tool.

Do not use `-json` as a CI pass/fail check: it can return exit code 0 with errors.
