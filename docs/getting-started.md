# Get started

[Documentation](README.md) · [Language rules](rules.md) · [Collection API](../pkg/im/readme.md)

In a Go module, add the library and install the checker:

```sh
go get github.com/lucasavila00/immutago@latest
go install github.com/lucasavila00/immutago/cmd/immutago@latest
```

Save this as `main.go`:

```go
package main

import "github.com/lucasavila00/immutago/pkg/im"

type User struct {
    im.Immut
    Name string
    Scores im.FrozenMap[string, int]
}

func main() {
    scores := map[string]int{"go": 10}
    frozen := im.FreezeMap(scores)
    user := &User{Name: "alice", Scores: frozen}
    shared := user
    user = &User{Name: "bob", Scores: frozen}

    next := shared.Scores.Clone()
    next["go"] = 20
    updated := im.FreezeMap(next)
    oldScore, _ := shared.Scores.Get("go")
    newScore, _ := updated.Get("go")
    println(shared.Name, user.Name, oldScore, newScore)
}
```

Check and run it:

```sh
immutago ./...
go run .
```

The checker reports no errors. The program prints `alice bob 10 20`.
Rebinding `user` does not change the shared object. Freeze consumes the native
builder without a copy. Clone gives independent mutable container storage.

Use `im.ImmutNoRef` when references must not be stored or returned. Both markers
prohibit field writes from construction. See the [complete rules](rules.md)
for pointer targets, calls, statements, and builder operations.

## Project checks

Run `immutago ./...` in CI for each build configuration you ship. Dependencies
are checked too. An ordinary Go build does not run the analyzer.

For vet integration, use `go vet -vettool=/absolute/path/to/immutago ./...`.
The normal CLI fails on diagnostics. Its standard `-json` mode can exit
successfully with findings; do not use that exit status alone as a check.

An unsupported form is an error. Follow its correction and the allowed-form
list. There are no suppressions or caller trust annotations. Freeze has no
runtime ownership checks; see the [trusted boundary](arch/validation.md#trusted-code).
