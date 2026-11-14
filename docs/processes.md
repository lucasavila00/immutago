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

## Trusted dependencies

A process can need a logger, an HTTP client, a factory function, or a test
fake. These values are mutable, so they are not allowed stored types. Wrap each
one in `proc.UncheckedDependency[T]` with `proc.Trust`:

```go
type FetcherArgs struct {
    im.Shared
    BaseURL string
    Log     proc.UncheckedDependency[*slog.Logger]
    Client  proc.UncheckedDependency[*http.Client]
}

func (f *Fetcher) Run(ctx context.Context) {
    f.Args().Log.Value().Info("start", "url", f.Args().BaseURL)
}

proc.Start[Fetcher](root, FetcherArgs{
    BaseURL: "https://example.com",
    Log:     proc.Trust(logger),
    Client:  proc.Trust(http.DefaultClient),
})
```

`proc.Trust(v)` is a promise from the caller: several processes can use `v` at
the same time. The value is not immutable. The checker does not prove the
promise. It only makes sure that the promise is written in the code, so that
reviewers can find each `proc.Trust` call. `proc.UncheckedDependency` is the only way to
give a process mutable data that the checker does not prove.

A false promise is a data race that the checker cannot find. Without
`proc.Trust`, the checker rejects a map in the start arguments. With it, the
checker accepts the map, as in the
[trusted map](../tests/testdata/accepted/process/trusted_map.go) example:

```go
proc.Start[Reporter](children, ReporterArgs{Seen: proc.Trust(c.seen)}) // accepted; c.seen is still written: a data race
```

Wrap only values that are safe for concurrent use, for example `*slog.Logger`,
`*http.Client`, or a function that does not change shared memory. The
[dependencies](../tests/testdata/accepted/process/dependencies.go) example
shows a logger, a client, a factory function, and a test fake.

## Allowed forms

The [config loader](../tests/testdata/accepted/process/config_loader.go) and
[limiter](../tests/testdata/accepted/process/limiter.go) examples show Owner fields
and PIDs in start arguments. The
[worker pool](../tests/testdata/accepted/process/worker_pool.go) example starts
children under the supervisor of its run. The
[dispatcher](../tests/testdata/accepted/process/dispatcher.go) example starts
and stops groups of children from its methods during one run. The [waker](../tests/testdata/accepted/process/waker.go) example shows `Init`. The tables use those example types, and the
`FetcherArgs` type from [trusted dependencies](#trusted-dependencies).

These allowed forms are complete. Other uses of a process, its type name, a
process pointer from `Proc()`, or a `proc.UncheckedDependency` are rejected. Parentheses preserve these forms.

| Allowed form | Example |
| --- | --- |
| Named, non-generic struct with one `proc.Process[A]` embedded by value; `A` is an allowed stored type, a `proc.UncheckedDependency[T]`, or a struct with exactly one `im` marker whose fields are allowed stored types or named `proc.UncheckedDependency[T]` fields; the other fields are ordinary Go without protected storage, or `im.Owner` fields | `type ConfigLoader struct { proc.Process[LoaderArgs]; mu sync.Mutex; current im.Owner[Config] }` |
| Process without arguments | `type Cache struct { proc.Process[proc.None]; hits int }` |
| Declare the entry point on the process | `func (l *ConfigLoader) Run(ctx context.Context) { ... }` (no results) |
| Declare an initializer on the process; the runtime calls it before the value is reachable | `func (w *Waker) Init() { w.wake = make(chan struct{}, 1) }` (no parameters or results) |
| Method with a pointer receiver | `func (l *ConfigLoader) Version() int { return l.version }` |
| Exported method other than `Run` and `Init` whose parameters and results are allowed stored types | `func (l *ConfigLoader) Limit(key string) int`, `func (c *Cache) Latest() Entry` |
| In a method, select a field of the receiver other than the marker | `l.version++`, `l.mu.Lock()`, `defer l.mu.Unlock()`, `l.current.Store(&Config{})` |
| In a method, call a method of the receiver directly and synchronously, other than `Run` and `Init` | `l.publish()`, `l.Lock()` for an embedded `sync.Mutex` |
| In a method, read the arguments with a direct call | `args := l.Args()`, `l.Args().Loader.Proc().Limit(l.Args().Key)` |
| In a method, call `Supervisor()` directly and synchronously on the receiver, and pass the result directly as the first argument of a direct, synchronous `proc.Start`, `proc.StartReserved`, or `proc.StartSupervisor` call | `proc.Start[Worker](p.Supervisor(), WorkerArgs{ID: i})`, `proc.StartReserved(p.Supervisor(), spare, args)`, `proc.StartSupervisor(p.Supervisor())` |
| Keep, store, start under, and stop the result of `proc.StartSupervisor(p.Supervisor())`; it is an ordinary `*proc.Supervisor` | `sub := proc.StartSupervisor(d.Supervisor()); d.groups[name] = group{sub: sub}`, later `<-g.sub.Stop()` |
| In a method, bind `Supervisor()` to one new local variable with `:=`, and use that variable only as the first argument of a direct, synchronous `proc.Start`, `proc.StartReserved`, or `proc.StartSupervisor` call in the same method, not in a function literal | `sup := p.Supervisor(); for i := range p.Args().Workers { proc.Start[Worker](sup, WorkerArgs{ID: i}) }` |
| Call an exported method other than `Run` and `Init`, declared on the process, directly and synchronously on `pid.Proc()` | `v := loader.Proc().Version()`, `(loader.Proc()).Version()` |
| Call through a PID in a field of a marked value | `peers.Loader.Proc().Version()` |
| Keep, copy, store, pass, or return a PID | `type Peers struct { im.Shared; Loader proc.PID[ConfigLoader] }` |
| Call through a PID in a function literal | `defer func() { limiter.Proc().Take() }()` |
| Start a process with its arguments | `loader := proc.Start[ConfigLoader](root, LoaderArgs{Path: "config.json"})` |
| Reserve a PID and start it later, for processes that need each other's PID | `pong := proc.Reserve[Pong](); ping := proc.Start[Ping](root, pong); proc.StartReserved(root, pong, ping)` |
| Wrap a value in a dependency, anywhere | `log := proc.Trust(logger)`, `FetcherArgs{Log: proc.Trust(logger)}` |
| Declare a named `proc.UncheckedDependency[T]` field in a struct with one `im` marker; `T` has no protected storage | `type FetcherArgs struct { im.Shared; Log proc.UncheckedDependency[*slog.Logger] }` |
| Use a `proc.UncheckedDependency[T]` or a struct with a dependency field as a local value, an ordinary function parameter or result, or a start argument | `args := FetcherArgs{Log: log}`, `proc.Start[Fetcher](root, args)`, `proc.Start[Tracer](root, proc.Trust(logger))` |
| In a method, call `Value` directly and synchronously on `l.Args()` or on one field of `l.Args()`, where `l` is the receiver | `f.Args().Log.Value().Info("start")`, `log := f.Args().Log.Value()`, `t.Args().Value()` for `proc.Process[proc.UncheckedDependency[*slog.Logger]]` |
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
| `waker.Proc().Init()`, `w.Init()` | Only the runtime calls `Init`. |
| `func (w *Waker) Init() error`, `func (w Waker) Init()`, a promoted `Init` | Declare `func (w *Waker) Init()`. |
| `loader.Proc().publish()`, `loader.Proc().Lock()` for an embedded `sync.Mutex` | Declare an exported method on the process. |
| `f := loader.Proc().Version`, `get := loader.Proc` | `v := loader.Proc().Version()` |
| `go loader.Proc().Version()`, `defer loader.Proc().Version()` | `go func() { loader.Proc().Version() }()` |
| In a method: `other := l`, `use(l)`, `l == nil`, `*l = ConfigLoader{}` | Select a field or call a method. |
| In a method: `go l.publish()`, `defer l.publish()`, `f := l.publish` | Call `l.publish()` directly; use `defer l.mu.Unlock()` for field cleanup. |
| In a method: `func() { l.version++ }` | Move the work into a method. |
| In a method: `l.Process`, `&l.Process`, `f := l.Args` | `args := l.Args()` |
| In a method: `p.sup = p.Supervisor()`, `saved = p.Supervisor()`, `keep(p.Supervisor())`, `_ = p.Supervisor()` | `proc.Start[Worker](p.Supervisor(), args)` |
| In a method: `p.Supervisor().Stop()`, `sup := p.Supervisor(); <-sup.Stop()` | Stop a nested supervisor: `sub := proc.StartSupervisor(p.Supervisor())`, later `<-sub.Stop()`. The runtime stops the supervisor of the run. |
| In a method: `get := p.Supervisor`, `go p.Supervisor()`, `go proc.Start[Worker](p.Supervisor(), args)`, `defer proc.StartSupervisor(sup)` | Call `proc.Start[Worker](p.Supervisor(), args)` directly. |
| In a method: `func() { proc.Start[Worker](p.Supervisor(), args) }()`, `sup := p.Supervisor(); go func() { proc.Start[Worker](sup, args) }()` | Start the child in the method, outside the function literal. |
| In a method: `var sup = p.Supervisor()`, `sup = p.Supervisor()`, `sup, n := p.Supervisor(), 1`, `other := sup`, `keep(sup)` | `sup := p.Supervisor()`, then `proc.Start[Worker](sup, args)` |
| `func (p *WorkerPool) owned() *proc.Supervisor { return p.Supervisor() }` | Return `proc.StartSupervisor(p.Supervisor())`, or start the children in the method. |
| `pool.Proc().Supervisor()`, `p.Process.Supervisor()`, `(*proc.Process[PoolArgs]).Supervisor` | A process starts children only under its own run, in its own methods. |
| `&ConfigLoader{}`, `var l ConfigLoader`, `new(ConfigLoader)` | `proc.Start[ConfigLoader](root, args)` |
| `func use(l *ConfigLoader)`, `type L = ConfigLoader`, `type Outer struct { ConfigLoader }`, `(*ConfigLoader).Version` | Pass a `proc.PID[ConfigLoader]`. |
| `proc.Start[ConfigLoader, LoaderArgs](root, args)` | `proc.Start[ConfigLoader](root, args)` |
| `proc.PID[int]`, `proc.Reserve[Plain]()`, `func f[T any](p proc.PID[T])` | Use a concrete process type. |
| `type Bad struct { proc.Process[map[string]int] }`, `proc.Process[error]` | Use an allowed argument type, for example `proc.Process[LoaderArgs]`. |
| `type Bad[T any] struct { proc.Process[T] }`, `var x struct{ proc.Process[proc.None] }` | Declare a named, non-generic struct. |
| `type Bad struct { proc.Process[proc.None]; im.Shared }`, `M proc.Process[proc.None]`, `*proc.Process[proc.None]` | Embed one `proc.Process[A]` by value and no `im` marker. |
| `type Bad struct { proc.Process[proc.None]; limits im.FrozenMap[string, int] }` | Put immutable data in the arguments, or publish it through an `im.Owner` field. |
| `type ReporterArgs struct { im.Shared; Seen map[string]int }` | `Seen proc.UncheckedDependency[map[string]int]` with `proc.Trust`, only if the map is safe for concurrent use. |
| `type Fetcher struct { proc.Process[FetcherArgs]; log proc.UncheckedDependency[*slog.Logger] }`, `type deps struct { log proc.UncheckedDependency[*slog.Logger] }` | Keep the dependency in the arguments; in a method, `log := f.Args().Log.Value()`. A process field of type `*slog.Logger` is ordinary Go. |
| `func (f *Fetcher) Log() proc.UncheckedDependency[*slog.Logger]`, `func (f *Fetcher) Arguments() FetcherArgs` | Return immutable data. |
| `im.Owner[State]`, `im.FrozenSlice[proc.UncheckedDependency[*slog.Logger]]`, `im.FrozenSlice[FetcherArgs]`, where `State` has a dependency field | Keep dependencies only in start arguments. |
| `type Envelope struct { im.Shared; Args FetcherArgs }`, `Args *FetcherArgs`, `Logs [2]proc.UncheckedDependency[*slog.Logger]` | Use `FetcherArgs` directly as `A` in `proc.Process[A]`. |
| `type Args struct { im.Shared; proc.UncheckedDependency[*slog.Logger] }` (embedded) | Name the field: `Log proc.UncheckedDependency[*slog.Logger]`. |
| `proc.Trust(&Snapshot{})`, where `Snapshot` embeds `im.Shared` | Put the immutable value in a field of the arguments. |
| `args := f.Args(); args.Log.Value()`, `func use(d proc.UncheckedDependency[*slog.Logger]) { d.Value() }`, `get := f.Args().Log.Value`, `go f.Args().Log.Value()` | `f.Args().Log.Value()` in a process method. |

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
- In a method, `p.Supervisor()` returns the supervisor of the current run of
  that process value. The first call for the value creates it. Children that a
  method starts before `Run` begins start with the run. When `Run` returns or
  panics, also after `Stop`, the runtime stops the children of this supervisor,
  last started first, and waits for them. Then the run ends: the restart delay
  starts, or `Stop` continues. After the run ends, `p.Supervisor()` returns a
  stopped supervisor, and processes started under it do not run. A new run
  uses a new process value, so it starts with no children. A PID of an old
  child keeps its last value, but that child does not run again.
  `StartReserved` panics on a PID from an earlier run, so reserve the PIDs of
  children in the same run. If a child passes its restart limit, the run
  context ends and the run fails with the panic value of the child. The
  supervisor of the process counts this failure as a panic of the process.
- To start and stop single children during one run, give each child its own
  nested supervisor: `sub := proc.StartSupervisor(p.Supervisor())`, then
  `proc.Start[Worker](sub, args)`. Keep `sub` in a field, and call
  `<-sub.Stop()` from any method to stop that child. The stopped nested
  supervisor is removed from the supervisor of the run.
- `proc.StartSupervisor(parent)` nests a supervisor. `Stop` begins stopping;
  its channel closes when all children have stopped, last started first.
  When a nested supervisor's own Stop channel closes, its parent has removed
  its child entry.
  Cancellation is cooperative: `Run` must observe its context. Stop does not
  wait for direct method calls, and methods must synchronize their shared state.
- `proctest.NewRoot(t)` fails a test whose process tree does not stop in 5 s.
