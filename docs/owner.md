# Owner

`im.Owner[T]` wraps Go's `atomic.Pointer[T]`. Load and Store are atomic pointer
operations; the checker adds rules for the objects they access.

| Runtime operation | Checker rule |
| --- | --- |
| `Store` replaces the pointer without copying the object | T must embed `im.Scoped`; supply a new object. |
| `Load` returns the current pointer, or nil before the first Store | Read through a local pointer; do not store it in a field or global, capture it, or return it. |
| Readers can keep using an old pointer after Store replaces it | The old object stays read-only. |

Go's GC keeps loaded objects alive while readers use them. Load and Store can
run in different goroutines. For example, update the request limit while requests
read the current configuration:

```go
package main

import "github.com/lucasavila00/immutago/pkg/im"

type Config struct {
    im.Scoped
    Limit int
}

var current im.Owner[Config]

func allowed(used int) bool {
    config := current.Load()
    if config == nil {
        return false
    }
    return used < config.Limit
}

func main() {
    current.Store(&Config{Limit: 100})
    println(allowed(80)) // true
    current.Store(&Config{Limit: 50})
    println(allowed(80)) // false
}
```

## Allowed forms — exhaustive

| Form | Example |
| --- | --- |
| Package Owner variable without an initializer | `var current im.Owner[Config]` |
| Named Owner field in a struct without an im marker | `type State struct { Current im.Owner[Config] }` |
| Local or package variable of that struct without an initializer | `var state State` |
| New struct pointer in a local variable | `state := &State{}` or `state := new(State)` |
| Copy that pointer to a local variable | `other := state` |
| Pass that pointer to a named, non-generic function with the exact fixed parameter type | `publish(state)` where `func publish(s *State)` |
| Store in a package variable or direct field, with exactly `&T{...}` or `new(T)`, as a direct call statement | `current.Store(&Config{Limit: 1})`, `state.Current.Store(new(Config))` |
| Load from a package variable or direct field into one local pointer | `config := current.Load()`, `var config = state.Current.Load()`, `config = current.Load()` |
| Package-qualified calls | `config := settings.Current.Load()` |
| Parentheses around an allowed expression | `config := (current).Load()` |

Loaded pointers use the [Scoped forms](scoped.md).
[Supervised services](service-forms.md#inside-a-service-method) have separate
rules for methods that access Owner fields.

A named publisher can run in a goroutine:

```go
func publish(s *State) { s.Current.Store(&Config{Limit: 1}) }

func start() {
    state := &State{}
    go publish(state)
}
```

## Rejected examples — non-exhaustive

| Rejected form | Replacement |
| --- | --- |
| `im.Owner[SharedConfig]`, where SharedConfig embeds `im.Shared` | Use `im.Owner[Config]` with `im.Scoped`. |
| `copy := state.Current` or `copy := *state` | Pass the struct pointer; do not copy the atomic pointer or its containing struct. |
| `current.Store(config)` | `current.Store(&Config{Limit: config.Limit})` |
| `return current.Load()` | `config := current.Load(); return *config` with a Config result. |
| `retain(state)` where retain takes `...*State` | Use a fixed `*State` parameter. |
| Store state in a field, slice, map, global, or closure; return state | Keep the pointer local or pass it to a named function. |
| Pass `&state.Current` as `*im.Owner[Config]` | Pass `state` as `*State`. |
| `go current.Store(&Config{})` or `defer current.Load()` | Call directly. Use `go publish(state)` for a background publisher. |
| `read := current.Load` | `config := current.Load()` |
| `current.Load().Limit = 2` | `current.Store(&Config{Limit: 2})` |
| `type Outer struct { Value State }` | Keep the Owner field directly in State. |
