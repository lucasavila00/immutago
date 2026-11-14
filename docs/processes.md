# Supervised processes

Use `proc` for a service that runs in the background, such as a config loader,
cache refresher, or rate limiter. Start it with immutable arguments, keep its
PID, and call its exported methods directly. Those methods take and return
[allowed stored types](common-forms.md#markers-and-stored-types), including PIDs.

The analyzer checks four rules: fields are selected through the receiver in its
methods; start arguments and exported method parameters/results use allowed
stored types; process pointers are used only for the listed direct calls; and
only `Start` or `StartReserved` creates a process.

The existing immutable-field and Owner rules still apply. In particular, mutable
process structs publish protected data through Owner fields.

Method bodies are ordinary Go. These checks do not isolate mutable state:
`shared = &p.count` or `p.table = globalMap` can share memory through globals or
helpers. Methods must follow their API contracts and synchronize shared state.

```go
package main

import (
    "context"
    "fmt"
    "sync"

    "github.com/lucasavila00/immutago/pkg/proc"
)

type Counter struct {
    proc.Process[int]
    mu sync.Mutex
    count int
}

func (*Counter) Run(ctx context.Context) { <-ctx.Done() }

func (c *Counter) Add() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count += c.Args()
    return c.count
}

func main() {
    root := proc.NewRoot()
    counter := proc.Start[Counter](root, 1)
    fmt.Println(counter.Proc().Add()) // 1
    <-root.Stop()
}
```

The [config loader](../tests/testdata/accepted/process/config_loader.go) and
[limiter](../tests/testdata/accepted/process/limiter.go) examples show Owner fields
and PIDs in start arguments. The tables use those example types.

These allowed forms are complete. Other uses of a process, its type name, or a
process pointer from `Proc()` are rejected. Parentheses preserve these forms.

| Allowed form | Example |
| --- | --- |
| Named, non-generic struct with one `proc.Process[A]` embedded by value; `A` is an allowed stored type; the other fields are ordinary Go without protected storage, or `im.Owner` fields | `type ConfigLoader struct { proc.Process[LoaderArgs]; mu sync.Mutex; current im.Owner[Config] }` |
| Process without arguments | `type Cache struct { proc.Process[proc.None]; hits int }` |
| Declare the entry point on the process | `func (l *ConfigLoader) Run(ctx context.Context) { ... }` (no results) |
| Method with a pointer receiver | `func (l *ConfigLoader) Version() int { return l.version }` |
| Exported method other than `Run` whose parameters and results are allowed stored types | `func (l *ConfigLoader) Limit(key string) int`, `func (c *Cache) Latest() Entry` |
| In a method, select a field of the receiver other than the marker | `l.version++`, `l.mu.Lock()`, `defer l.mu.Unlock()`, `l.current.Store(&Config{})` |
| In a method, call a method of the receiver directly and synchronously, other than `Run` | `l.publish()`, `l.Lock()` for an embedded `sync.Mutex` |
| In a method, read the arguments with a direct call | `args := l.Args()`, `l.Args().Loader.Proc().Limit(l.Args().Key)` |
| Call an exported method other than `Run`, declared on the process, directly and synchronously on `pid.Proc()` | `v := loader.Proc().Version()`, `(loader.Proc()).Version()` |
| Call through a PID in a field of a marked value | `peers.Loader.Proc().Version()` |
| Keep, copy, store, pass, or return a PID | `type Peers struct { im.Shared; Loader proc.PID[ConfigLoader] }` |
| Call through a PID in a function literal | `defer func() { limiter.Proc().Take() }()` |
| Start a process with its arguments | `loader := proc.Start[ConfigLoader](root, LoaderArgs{Path: "config.json"})` |
| Reserve a PID and start it later, for processes that need each other's PID | `pong := proc.Reserve[Pong](); ping := proc.Start[Ping](root, pong); proc.StartReserved(root, pong, ping)` |
| Name a process type in its declaration, a pointer receiver, or the single type argument of `proc.Start`, `proc.StartReserved`, `proc.Reserve`, or `proc.PID` | `proc.PID[ConfigLoader]`, `proc.Reserve[Pong]()` |

Rejected examples are not exhaustive:

| Rejected example | Allowed replacement |
| --- | --- |
| `func (l *ConfigLoader) Table() map[string]int` | Publish an object with an `im.FrozenMap`. |
| `func (l *ConfigLoader) Each(visit func(string))` | Return immutable data. |
| `func (l *ConfigLoader) Load() error` | Return a defined scalar status; `error` is an interface. |
| `func (l *ConfigLoader) Keys(keys ...string)` | Take an `im.FrozenSlice[string]`. |
| Missing `Run`, a promoted `Run`, or `Run(int)` | Declare `func (l *ConfigLoader) Run(ctx context.Context)`. |
| `func (l ConfigLoader) Version() int` | `func (l *ConfigLoader) Version() int` |
| `p := loader.Proc()`, `saved = loader.Proc()`, `use(loader.Proc())`, `loader.Proc() == nil` | Keep the PID; call the method directly. |
| `loader.Proc().version`, `loader.Proc().Args()` | Add an exported method. |
| `loader.Proc().Run(ctx)`, `l.Run(ctx)` | Only the supervisor calls `Run`. |
| `loader.Proc().publish()`, `loader.Proc().Lock()` for an embedded `sync.Mutex` | Declare an exported method on the process. |
| `f := loader.Proc().Version`, `get := loader.Proc` | `v := loader.Proc().Version()` |
| `go loader.Proc().Version()`, `defer loader.Proc().Version()` | `go func() { loader.Proc().Version() }()` |
| In a method: `other := l`, `use(l)`, `l == nil`, `*l = ConfigLoader{}` | Select a field or call a method. |
| In a method: `go l.publish()`, `defer l.publish()`, `f := l.publish` | Call `l.publish()` directly; use `defer l.mu.Unlock()` for field cleanup. |
| In a method: `func() { l.version++ }` | Move the work into a method. |
| In a method: `l.Process`, `&l.Process`, `f := l.Args` | `args := l.Args()` |
| `&ConfigLoader{}`, `var l ConfigLoader`, `new(ConfigLoader)` | `proc.Start[ConfigLoader](root, args)` |
| `func use(l *ConfigLoader)`, `type L = ConfigLoader`, `type Outer struct { ConfigLoader }`, `(*ConfigLoader).Version` | Pass a `proc.PID[ConfigLoader]`. |
| `proc.Start[ConfigLoader, LoaderArgs](root, args)` | `proc.Start[ConfigLoader](root, args)` |
| `proc.PID[int]`, `proc.Reserve[Plain]()`, `func f[T any](p proc.PID[T])` | Use a concrete process type. |
| `type Bad struct { proc.Process[map[string]int] }`, `proc.Process[error]` | Use an allowed argument type, for example `proc.Process[LoaderArgs]`. |
| `type Bad[T any] struct { proc.Process[T] }`, `var x struct{ proc.Process[proc.None] }` | Declare a named, non-generic struct. |
| `type Bad struct { proc.Process[proc.None]; im.Shared }`, `M proc.Process[proc.None]`, `*proc.Process[proc.None]` | Embed one `proc.Process[A]` by value and no `im` marker. |
| `type Bad struct { proc.Process[proc.None]; limits im.FrozenMap[string, int] }` | Put immutable data in the arguments, or publish it through an `im.Owner` field. |

The runtime behavior is not part of the proof:

- `Start` creates a new process value, gives it the arguments, stores it in the
  PID, and runs `Run` in a new goroutine. The PID is started when `Start` returns.
  `Proc` is one atomic load. It panics with `proc: PID not started` before
  `StartReserved` starts a reserved PID. `StartReserved` panics if the PID is
  already started.
- Each run gets a context that ends with that run. If `Run` returns, the process
  ends. If `Run` panics, the supervisor waits 1 s, doubling the delay for each
  restart in the last 5 minutes. It then creates a fresh value from the same
  arguments, publishes it through the PID, and calls `Run`. `im.Owner` fields
  start empty again. Calls during the delay use the old value; calls already
  in progress finish on that value. Reserving PIDs is not a group-start barrier:
  a call to a peer that has not started can panic and cause this retry.
- A sixth panic from the same child in 5 minutes stops the other children of the supervisor in
  reverse start order and escalates to the parent supervisor, which restarts
  the subtree with the same rule. At the root, the last panic value panics
  again and the program exits.
- `proc.StartSupervisor(parent)` nests a supervisor. `Stop` begins stopping;
  its channel closes when all children have stopped, last started first.
  Cancellation is cooperative: `Run` must observe its context. Stop does not
  wait for direct method calls, and methods must synchronize their shared state.
- `proctest.NewRoot(t)` fails a test whose process tree does not stop in 5 s.

