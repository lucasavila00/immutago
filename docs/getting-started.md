# Get started

## Install

In a Go module, add the library and install the checker:

```sh
go get github.com/lucasavila00/immutago@latest
go install github.com/lucasavila00/immutago/cmd/immutago@latest
```

## Example: immutable user data

This program creates a user with a read-only score map. The checker will reject
code that changes the user's fields or the frozen map.

Save this as `main.go`:

```go
package main

import "github.com/lucasavila00/immutago/pkg/im"

type User struct {
    im.Shared
    Name   im.FrozenString
    Scores im.FrozenMap[im.FrozenString, int]
}

func main() {
    scores := map[im.FrozenString]int{im.FreezeString("go"): 10}
    frozen := im.FreezeMap(scores)
    user := &User{Name: im.FreezeString("alice"), Scores: frozen}
    score, _ := user.Scores.Get(im.FreezeString("go"))
    println(user.Name.Clone(), score)
}
```

Check and run it:

```sh
immutago ./...
go run .
```

The program prints `alice 10`.

- `im.Shared` is an empty embedded marker. The checker rejects field writes and
  permits sharing the object's pointer.
- `FreezeMap` reuses the Go map without copying its entries. After this call,
  read through the frozen map, not `scores`.
- `FreezeString` copies native text; `Clone` returns an independent native string.
- `Get` returns the score and whether the key exists.

For example, adding `user.Name = im.FreezeString("bob")` makes `immutago ./...` report a write
to an immutable object. See the [language guide](rules.md) for the complete forms.

## Check in CI

```sh
immutago ./...
```

- Errors make the command fail.
- Run it for each platform and set of build tags you use.
- `go build` does not run the checker.

Or use `go vet`:

```sh
go vet -vettool=/absolute/path/to/immutago ./...
```

Do not use `-json` as a CI pass/fail check: it can return exit code 0 with errors.
