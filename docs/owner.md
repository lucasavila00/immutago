# Owner

Use `im.Owner[T]` when readers need the current version of some data while a
publisher replaces it. `T` must be a Scoped struct. Store creates no change in
an object that a reader already loaded.

Here, each request reads one configuration version. A later Store affects later
Loads, while an active reader can finish with its earlier version:

```go
package main

import "github.com/lucasavila00/immutago/pkg/im"

type Config struct {
    im.Scoped
    Limit int
}

type Service struct {
    Current im.Owner[Config]
}

func configure(service *Service, limit int) {
    service.Current.Store(&Config{Limit: limit})
}

func allowed(service *Service, used int) bool {
    config := service.Current.Load()
    if config == nil {
        return false
    }
    return used < config.Limit
}

func main() {
    service := &Service{}
    configure(service, 100)
    println(allowed(service, 80)) // true
    configure(service, 50)
    println(allowed(service, 80)) // false
}
```

Load and Store can run in different goroutines. Load returns nil before the
first Store. Pass a newly created object to Store; a pointer from Load follows
the [Scoped rules](scoped.md).

The reference below uses `ScopedPoint` for a Scoped struct with `X int`, and
`Loader` for a struct with `Current im.Owner[ScopedPoint]`.

## Allowed forms

An Owner field or variable is a **slot**. A struct with a slot field, such as
`Loader`, is a **holder**. These allowed forms are complete; other slot and
holder operations are rejected.
Parentheses are allowed. An exported Owner variable can also be used through
its package name, such as `state.Current.Load()`.

| Allowed form | Example |
| --- | --- |
| Named Owner field in a struct without an im marker | `type Loader struct { Current im.Owner[ScopedPoint] }` |
| Declare a package Owner variable without an initializer | `var current im.Owner[ScopedPoint]` |
| Declare a local or package holder variable without an initializer | `var loader Loader` |
| Create a holder and keep its pointer in a local variable | `loader := &Loader{}` or `loader := new(Loader)` |
| Copy a holder pointer to one local variable | `other := loader` |
| Pass a holder pointer to a direct, concrete function | `publish(loader)` where `func publish(l *Loader)` |
| Store a fresh object in a direct slot field or package slot | `loader.Current.Store(&ScopedPoint{X: 1})`, `current.Store(new(ScopedPoint))` |
| Load into one local pointer | `point := loader.Current.Load()`, `var point = current.Load()`, `point = current.Load()` |

A named function can publish in a background goroutine:

```go
func publish(l *Loader) { l.Current.Store(&ScopedPoint{X: 1}) }

func start() {
    loader := &Loader{}
    go publish(loader)
}
```

[Processes](processes.md) have separate rules for methods that use Owner fields.
Do not copy an Owner or a struct that contains one; pass a pointer to the struct.

Keep `*Loader` pointers in local variables. Pass them to named functions with
a parameter such as `l *Loader`. Do not save them in fields, slices, maps,
package variables, or closures, and do not return them. Do not use generic or
variadic functions for these calls. Pass `*Loader`, not `*im.Owner[T]`.

A pointer returned by Load follows the [Scoped rules](scoped.md). Save the
result in a local variable before use. For `im.Owner[T]`, call Store with
`&T{...}` or `new(T)`, not an existing pointer. Call Load and Store normally,
without `go` or `defer`.

## Rejected examples

These examples are not exhaustive.

| Rejected form | Allowed replacement |
| --- | --- |
| `im.Owner[SharedPoint]` | Use `im.Owner[ScopedPoint]`. |
| `copy := loader.Current` or `copy := *loader` | Keep and pass `loader` as a pointer. |
| `loader.Current.Store(point)` | `loader.Current.Store(&ScopedPoint{X: point.X})` |
| `return loader.Current.Load()` | `point := loader.Current.Load(); return *point` with a value result. |
| `retain(loader)` where `retain` takes `...*Loader` | Use a direct function with a fixed `*Loader` parameter. |
| `var saved *Loader` at package scope | Keep the pointer in a local variable. |
| `read := loader.Current.Load` | `point := loader.Current.Load()` |
| `loader.Current.Load().X = 2` | `loader.Current.Store(&ScopedPoint{X: 2})` |
| `type Outer struct { Value Loader }` | Keep the slot field in the unmarked holder. |
