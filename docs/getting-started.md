# Get started

This program creates a user with a read-only score map. The checker will reject
code that changes the user's fields or the frozen map.

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

The program prints `alice 10`.

- `im.Shared` makes the struct's fields read-only and permits sharing its pointer.
- `FreezeMap` reuses the Go map without copying its entries. After this call,
  read through the frozen map, not `scores`.
- `Get` returns the score and whether the key exists.

To see the checker reject a change, add `user.Name = "bob"` after the println
and run `immutago ./...` again. It reports a write to an immutable object.
Remove that line before continuing.

Choose [Shared](shared.md) to save or return pointers. Choose [Scoped](scoped.md)
to keep pointers local, or [Owner](owner.md) to publish new Scoped objects.
The [language guide](rules.md) lists the complete allowed forms.

## Check in CI

```sh
immutago ./...
```

The command fails if it finds errors. `go build` does not run the checker.
Run it for each platform and set of build tags you use.

Or use `go vet`:

```sh
go vet -vettool=/absolute/path/to/immutago ./...
```

Do not use `-json` as a CI pass/fail check: it can return exit code 0 with errors.
