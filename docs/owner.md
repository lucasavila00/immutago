# Owner

Use `im.Owner[T]` to replace data while readers finish with the previous version:

- `T` is a struct with `im.Scoped`.
- `Store` publishes a new object; existing readers keep the previous object.
- `Load` reads the current object, or returns nil before the first Store.
- Load and Store can run in different goroutines.

This example replaces the request limit:

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
    previous := service.Current.Load()
    println(allowed(service, 80)) // true
    configure(service, 50)
    println(allowed(service, 80)) // false
    println(previous.Limit) // 100: this reader keeps its earlier version.
}
```

## Allowed forms

An Owner field or variable is a **slot**. A struct with a slot field, such as
`Service`, is a **holder**. These allowed forms are complete; other slot and
holder operations are rejected.

- Parentheses preserve an allowed form.
- A package-qualified slot works: `state.Current.Load()`.
- Call Load and Store directly, without `go` or `defer`.
- Load results follow the [Scoped rules](scoped.md).

| Allowed form | Example |
| --- | --- |
| Named Owner field in a struct without an im marker | `type Service struct { Current im.Owner[Config] }` |
| Declare a package Owner variable without an initializer | `var current im.Owner[Config]` |
| Declare a local or package holder variable without an initializer | `var service Service` |
| Create a holder and keep its pointer in a local variable | `service := &Service{}` or `service := new(Service)` |
| Copy a holder pointer to one local variable | `other := service` |
| Pass a holder pointer to a direct, non-generic function with a fixed parameter | `publish(service)` where `func publish(l *Service)` |
| Store a fresh object in a direct slot field or package slot | `service.Current.Store(&Config{Limit: 1})`, `current.Store(new(Config))` |
| Load into one local pointer | `config := service.Current.Load()`, `var config = current.Load()`, `config = current.Load()` |

A named function can publish in a background goroutine:

```go
func publish(l *Service) { l.Current.Store(&Config{Limit: 1}) }

func start() {
    service := &Service{}
    go publish(service)
}
```

[Services](services.md) have separate rules for methods that use Owner fields.

## Rejected examples

These examples are not exhaustive.

| Rejected form | Allowed replacement |
| --- | --- |
| `im.Owner[SharedConfig]`, where SharedConfig embeds `im.Shared` | Use `im.Owner[Config]`. |
| `copy := service.Current` or `copy := *service` | Keep and pass `service` as a pointer. |
| `service.Current.Store(config)` | `service.Current.Store(&Config{Limit: config.Limit})` |
| `return service.Current.Load()` | `config := service.Current.Load(); return *config` with a value result. |
| `retain(service)` where `retain` takes `...*Service` | Use a direct function with a fixed `*Service` parameter. |
| `var saved *Service` at package scope | Keep the pointer in a local variable. |
| Store `service` in a field, slice, map, or closure; return `service` | Pass the local pointer to a named function. |
| Pass `&service.Current` as `*im.Owner[Config]` | Pass `service` as `*Service`. |
| `go service.Current.Store(&Config{})` or `defer service.Current.Load()` | Call Load and Store directly. A named publisher can run with `go publish(service)`. |
| `read := service.Current.Load` | `config := service.Current.Load()` |
| `service.Current.Load().Limit = 2` | `service.Current.Store(&Config{Limit: 2})` |
| `type Outer struct { Value Service }` | Keep the slot field in the unmarked holder. |
