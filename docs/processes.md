# Supervised processes

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

Methods use ordinary Go. Use locks where needed, as in `Add` above.

## Not checked

The checker controls what goes into and out of a process: start arguments,
method parameters and results, and process pointers. It does not check package
variables or the functions that use them. Two processes can share a mutable
global, and the checker accepts the program:

```go
var seen = map[string]int{}

type Counter struct{ proc.Process[proc.None] }

func (*Counter) Run(ctx context.Context) {
    for ctx.Err() == nil {
        seen["hits"]++ // Reader reads the same map: a data race
    }
}

type Reader struct{ proc.Process[proc.None] }

func (*Reader) Run(ctx context.Context) {
    for ctx.Err() == nil {
        _ = seen["hits"]
    }
}
```

`go test -race` finds this race; the checker does not. Keep shared state in one
process and publish it through an `im.Owner` field, or protect the global with a
lock.

## Allowed forms

The [config loader](../tests/testdata/accepted/process/config_loader.go) and
[limiter](../tests/testdata/accepted/process/limiter.go) examples show Owner fields
and PIDs in start arguments. The [fetcher](../tests/testdata/accepted/process/fetcher.go)
example shows ordinary arguments and `Init`. The tables use those example types.

These allowed forms are complete. Other uses of a process, its type name, or a
process pointer from `Proc()` are rejected. Parentheses preserve these forms.

| Allowed form | Example |
| --- | --- |
| Named, non-generic struct with one `proc.Process[A]` embedded by value; `A` is an allowed stored type, or ordinary Go without protected storage and without callbacks that carry it; the other fields are ordinary Go without protected storage, or `im.Owner` fields | `type ConfigLoader struct { proc.Process[LoaderArgs]; mu sync.Mutex; current im.Owner[Config] }` |
| Process without arguments | `type Cache struct { proc.Process[proc.None]; hits int }` |
| Ordinary arguments | `type FetcherArgs struct { Logger *slog.Logger; Client *http.Client; Loader proc.PID[ConfigLoader] }` |
| Declare the entry point on the process | `func (l *ConfigLoader) Run(ctx context.Context) { ... }` (no results) |
| Declare an initializer on the process; the runtime calls it before the value is reachable | `func (f *Fetcher) Init() { f.wake = make(chan struct{}, 1) }` (no parameters or results) |
| Method with a pointer receiver | `func (l *ConfigLoader) Version() int { return l.version }` |
| Exported method other than `Run` and `Init` whose parameters and results are allowed stored types | `func (l *ConfigLoader) Limit(key string) int`, `func (c *Cache) Latest() Entry` |
| In a method, select a field of the receiver other than the marker | `l.version++`, `l.mu.Lock()`, `defer l.mu.Unlock()`, `l.current.Store(&Config{})` |
| In a method, call a method of the receiver directly and synchronously, other than `Run` and `Init` | `l.publish()`, `l.Lock()` for an embedded `sync.Mutex` |
| In a method, read the arguments with a direct call | `args := l.Args()`, `l.Args().Loader.Proc().Limit(l.Args().Key)` |
| Call an exported method other than `Run` and `Init`, declared on the process, directly and synchronously on `pid.Proc()` | `v := loader.Proc().Version()`, `(loader.Proc()).Version()` |
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
| `fetcher.Proc().Init()`, `f.Init()` | Only the runtime calls `Init`. |
| `func (f *Fetcher) Init() error`, `func (f Fetcher) Init()`, a promoted `Init` | Declare `func (f *Fetcher) Init()`. |
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
| `type Bad struct { proc.Process[*Config] }`, `proc.Process[func(*Config)]` | Use an allowed stored type for protected storage, for example `proc.Process[LoaderArgs]`. |
| `type Bad[T any] struct { proc.Process[T] }`, `var x struct{ proc.Process[proc.None] }` | Declare a named, non-generic struct. |
| `type Bad struct { proc.Process[proc.None]; im.Shared }`, `M proc.Process[proc.None]`, `*proc.Process[proc.None]` | Embed one `proc.Process[A]` by value and no `im` marker. |
| `type Bad struct { proc.Process[proc.None]; limits im.FrozenMap[string, int] }` | Put immutable data in the arguments, or publish it through an `im.Owner` field. |

## Start, restart, and stop

- `Start` creates a new process value, gives it the arguments, calls `Init` if
  the process declares it, stores the value in the PID, and runs `Run` in a new
  goroutine. A panic in this first `Init` reaches the caller of `Start`. The PID is started when `Start` returns.
  `Proc` is one atomic load. It panics with `proc: PID not started` before
  `StartReserved` starts a reserved PID. `StartReserved` panics if the PID is
  already started.
- Each run gets a context that ends with that run. If `Run` returns, the process
  ends. If `Run` panics, the supervisor waits 1 s, doubling the delay for each
  restart in the last 5 minutes. It then creates a fresh value from the same
  arguments, calls `Init`, publishes it through the PID, and calls `Run`. `im.Owner` fields
  start empty again. Calls during the delay use the old value; calls already
  in progress finish on that value. Reserve does not wait for
  other processes to start. Calling a reserved PID before StartReserved can
  panic and cause a retry.
- A sixth panic from the same child in 5 minutes stops the other children of the supervisor in
  reverse start order and escalates to the parent supervisor, which restarts
  the subtree with the same rule. At the root, the last panic value panics
  again and the program exits.
- `proc.NewRoot(proc.OnPanic(report))` calls `report` with the value and the
  stack of each panic that a supervisor of the tree recovers, before the restart
  delay, for example to log it: `proc.OnPanic(func(value any, stack []byte) { log.Printf("%v\n%s", value, stack) })`.
- `proc.StartSupervisor(parent)` nests a supervisor. `Stop` begins stopping;
  its channel closes when all children have stopped, last started first.
  When a nested supervisor's own Stop channel closes, its parent has removed
  its child entry.
  Cancellation is cooperative: `Run` must observe its context. Stop does not
  wait for direct method calls, and methods must synchronize their shared state.
- `proctest.NewRoot(t)` fails a test whose process tree does not stop in 5 s.
