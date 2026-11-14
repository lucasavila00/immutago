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
    im.Shared
    Name   string
    Scores im.FrozenMap[string, int]
}

func main() {
    scores := map[string]int{"go": 10}
    frozen := im.FreezeMap(scores)
    user := &User{Name: "alice", Scores: frozen}
    score, _ := user.Scores.Get("go")
    println(user.Name, score)
}
```

Check and run it:

```sh
immutago ./...
go run .
```

The program prints `alice 10`. The checker makes `User` fields read-only.
`FreezeMap` transfers the map's storage without a copy; use `user.Scores` to
read it after Freeze.

Use `im.Scoped` when pointers must not be stored or returned. Both markers
prohibit field writes from construction. See the [complete rules](rules.md)
for pointer targets, calls, statements, and builder operations.

## Project checks

Run `immutago ./...` in CI for each build configuration you ship. Dependencies
are checked too. An ordinary Go build does not run the analyzer.

For vet integration, use `go vet -vettool=/absolute/path/to/immutago ./...`.
The normal CLI fails on diagnostics. Its standard `-json` mode can exit
successfully with findings; do not use that exit status alone as a check.
