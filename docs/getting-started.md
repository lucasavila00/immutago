# Get started

[Documentation](README.md) · [Language rules](rules.md) · [Collection API](../pkg/im/readme.md)

Immutago adds an analyzer and the `im` collection package to ordinary Go.
It does not change the compiler. An ordinary `go build` does not run the
immutability checks.

## Add it to a project

Use a Go toolchain that meets the version in [go.mod](../go.mod). In your Go
module, add the library and install the matching checker:

```sh
go get github.com/lucasavila00/immutago@latest
go install github.com/lucasavila00/immutago/cmd/immutago@latest
```

Put Go's binary installation directory on `PATH`. For repeatable builds, use
the same version for the library and checker and record that version in your
build configuration. From this repository, `make build` produces
`bin/immutago` (`bin/immutago.exe` on Windows).

## Build, borrow, and freeze

Save this complete example as `main.go`:

```go
package main

import "github.com/lucasavila00/immutago/pkg/im"

// User owns a name and read-only scores.
type User struct {
    im.Marker
    Name   string
    Scores im.FrozenMap[string, int]
}

// scoreCount reads a shared user without retaining its reference.
func scoreCount(user *User) int {
    return user.Scores.Len()
}

// main constructs and reads immutable storage.
func main() {
    scores := map[string]int{"go": 10}
    frozen := im.FreezeMap(scores)
    user := User{Name: "alice", Scores: frozen}

    p := &user
    count := scoreCount(p) // Last use of p.
    user.Name = "bob"     // The borrow has ended.

    next := user.Scores.Clone()
    next["go"] = 20
    updated := im.FreezeMap(next)

    oldScore, oldFound := user.Scores.Get("go")
    newScore, newFound := updated.Get("go")
    println(user.Name, count, oldScore, oldFound, newScore, newFound)
}
```

Check it, then run it:

```sh
immutago ./...
go run .
```

The analyzer reports no errors. The program prints:

```text
bob 1 10 true 20 true
```

`im.Marker` must be embedded by value. Each field must have an allowed immutable
type. `&user` acts like a Rust shared borrow, `&T`: reads are allowed, writes
through it are rejected. The borrow ends after its last use.

Freeze shares the native map's storage. After `im.FreezeMap(scores)`, any use of
`scores` is rejected. Clone creates independent mutable storage, so changing
`next` leaves `user.Scores` unchanged. See [costs](performance.md) before choosing
Clone for frequent updates.

For a slice, use the same sequence:

```go
values := []int{10, 20}
frozen := im.FreezeSlice(values)
first := frozen.At(0)
```

These are introductory examples, not the full language. The
[language rules](rules.md) list all supported reference, statement, and builder
forms. Rejected examples there are not exhaustive.

## Run checks in your build

Run `immutago ./...` from the consumer module to check its packages. The driver
also checks dependencies and reports relevant failures in imported code.
Run it for each build configuration that you ship; Go build tags and target
settings determine which files are loaded.

You can use the same binary with `go vet`:

```sh
go vet -vettool=/absolute/path/to/immutago ./...
```

Use the platform's absolute path to the binary. For automation, the normal CLI
reports failure when it finds errors. The standard driver's `-json` mode can
exit successfully with diagnostics, so do not use its exit status alone as a
pass/fail check.

## When a check fails

Read the source location, failed rule, and suggested correction. For example,
`next := p` copies a reference and is rejected. Use `next := *p` for an independent
value copy, or borrow the owner directly with `next := &user`.

An unsupported form is an error, even if a particular use seems safe. Rewrite
it to a listed form. There are no suppression directives or user trust
annotations. The checker enforces the rules; there is no additional alias
rule for callers to uphold by hand.

The checker trusts Go and the small `im` storage implementation. Freeze has no
runtime validation when used without the analyzer. The
[validation page](arch/validation.md#trusted-code) describes that boundary.
