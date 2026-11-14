# Owner

Use `im.Owner[T]` when readers need the current version of some data while a
publisher replaces it. `T` must be a Scoped struct. Store publishes a new object. It does not
change an object that a reader already loaded.

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
    previous := service.Current.Load()
    println(allowed(service, 80)) // true
    configure(service, 50)
    println(allowed(service, 80)) // false
    println(previous.Limit) // 100: this reader keeps its earlier version.
}
```

Load and Store can run in different goroutines. Load returns nil before the
first Store. Pass a newly created object to Store; a pointer from Load follows
the [Scoped rules](scoped.md).

## Allowed forms

An Owner field or variable is a **slot**. A struct with a slot field, such as
`Service`, is a **holder**. These allowed forms are complete; other slot and
holder operations are rejected.
Parentheses are allowed. An exported Owner variable can also be used through
its package name, such as `state.Current.Load()`.

| Allowed form | Example |
| --- | --- |
| Named Owner field in a struct without an im marker | `type Service struct { Current im.Owner[Config] }` |
| Declare a package Owner variable without an initializer | `var current im.Owner[Config]` |
| Declare a local or package holder variable without an initializer | `var service Service` |
| Create a holder and keep its pointer in a local variable | `service := &Service{}` or `service := new(Service)` |
| Copy a holder pointer to one local variable | `other := service` |
| Pass a holder pointer to a direct, concrete function | `publish(service)` where `func publish(l *Service)` |
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

[Processes](processes.md) have separate rules for methods that use Owner fields.
Do not copy an Owner or a struct that contains one; pass a pointer to the struct.

Keep `*Service` pointers in local variables. Pass them to named functions with
a parameter such as `l *Service`. Do not save them in fields, slices, maps,
package variables, or closures, and do not return them. Do not use generic or
variadic functions for these calls. Pass `*Service`, not `*im.Owner[T]`.

A pointer returned by Load follows the [Scoped rules](scoped.md). Save the
result in a local variable before use. For `im.Owner[T]`, call Store with
`&T{...}` or `new(T)`, not an existing pointer. Call Load and Store normally,
without `go` or `defer`.

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
| `read := service.Current.Load` | `config := service.Current.Load()` |
| `service.Current.Load().Limit = 2` | `service.Current.Store(&Config{Limit: 2})` |
| `type Outer struct { Value Service }` | Keep the slot field in the unmarked holder. |
