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

The checker also does not check the lifetime of PIDs. A process can reserve a
PID outside its run and start it under the supervisor of its run:

```go
type PoolArgs struct {
    im.Shared
    Spare proc.PID[Worker] // reserved by the caller
}

func (p *Pool) Run(ctx context.Context) {
    proc.StartReserved(p.Supervisor(), p.Args().Spare, WorkerArgs{})
    <-ctx.Done()
}
```

After a restart of `Pool`, the new run calls `StartReserved` on a PID that is
already started. It panics with `proc: PID already started` on each run, and
after six panics in 5 minutes the panic escalates to the parent supervisor. Call
`proc.Reserve` in the same method as `StartReserved`, as the
[dispatcher](../tests/testdata/accepted/process/dispatcher.go) does.

## Allowed forms

The [config loader](../tests/testdata/accepted/process/config_loader.go) and
[limiter](../tests/testdata/accepted/process/limiter.go) examples show Owner fields
and PIDs in start arguments. The
[worker pool](../tests/testdata/accepted/process/worker_pool.go) example starts
children under the supervisor of its run. The
[dispatcher](../tests/testdata/accepted/process/dispatcher.go) example starts
and stops groups of children from its methods during one run. The tables use those example types.

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
| In a method, call `Supervisor()` directly and synchronously on the receiver, and pass the result directly as the first argument of a direct, synchronous `proc.Start`, `proc.StartReserved`, or `proc.StartSupervisor` call | `proc.Start[Worker](p.Supervisor(), WorkerArgs{ID: i})`, `proc.StartReserved(p.Supervisor(), spare, args)`, `proc.StartSupervisor(p.Supervisor())` |
| Keep, store, start under, and stop the result of `proc.StartSupervisor(p.Supervisor())`; it is an ordinary `*proc.Supervisor` | `sub := proc.StartSupervisor(d.Supervisor()); d.groups[name] = group{sub: sub}`, later `<-g.sub.Stop()` |
| In a method, bind `Supervisor()` to one new local variable with `:=`, and use that variable only as the first argument of a direct, synchronous `proc.Start`, `proc.StartReserved`, or `proc.StartSupervisor` call in the same method, not in a function literal | `sup := p.Supervisor(); for i := range p.Args().Workers { proc.Start[Worker](sup, WorkerArgs{ID: i}) }` |
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

## Start, restart, and stop

- `Start` creates a new process value, gives it the arguments, stores it in the
  PID, and runs `Run` in a new goroutine. The PID is started when `Start` returns.
  `Proc` is one atomic load. It panics with `proc: PID not started` before
  `StartReserved` starts a reserved PID. `StartReserved` panics if the PID is
  already started.
- Each run gets a context that ends with that run. If `Run` returns, the process
  ends, and its supervisor removes it. If `Run` panics, the supervisor waits 1 s, doubling the delay for each
  restart in the last 5 minutes. It then creates a fresh value from the same
  arguments, publishes it through the PID, and calls `Run`. `im.Owner` fields
  start empty again. Calls during the delay use the old value; calls already
  in progress finish on that value. Reserve does not wait for
  other processes to start. Calling a reserved PID before StartReserved can
  panic and cause a retry.
- A sixth panic from the same child in 5 minutes stops the other children of
  the supervisor in reverse start order and escalates to the parent supervisor,
  which restarts the subtree with the same rule. At the root, the last panic value panics
  again and the program exits.
- In a method, `p.Supervisor()` returns the supervisor of the current run of
  that process value. The first call for the value creates it. Children that a
  method starts before `Run` begins start with the run.
- When the process stops, the runtime stops the children of this supervisor,
  last started first, and waits for them. Then it ends the `Run` context, so a
  child can call methods of its parent while the child stops. When `Run`
  returns or panics, the runtime stops the children and waits for them. Then
  the run ends: the restart delay starts, or `Stop` continues.
- After the run ends, `p.Supervisor()` returns a stopped supervisor. `Start`
  under it, or under a stored nested supervisor of that run, returns a PID
  whose `Run` never runs.
- If a child passes its restart limit, the run context ends and the run fails
  with the panic value of the child. The supervisor of the process counts this
  failure as a panic of the process. A group in a nested supervisor first
  restarts alone; if the group fails six times in 5 minutes, the run fails.
- A new run uses a new process value, so it starts with no children and empty
  fields. Children that methods started are gone too. `Run` must start them
  again from data that survives the restart: the start arguments, or a peer
  process, for example a config process with an `im.Owner`, whose PID is in the
  arguments. The [dispatcher](../tests/testdata/accepted/process/dispatcher.go)
  asks its config peer for the current variants in `Run`.
- After a restart of the parent, a PID of an old child keeps its old value, but
  that child does not run again. Do not publish child PIDs: do not store them
  in an `im.Owner`, send them to other processes, or return them from methods.
  Reach a child through its parent on each call.
- To start and stop single children during one run, give each child its own
  nested supervisor: `sub := proc.StartSupervisor(p.Supervisor())`, then
  `proc.Start[Worker](sub, args)`. Keep `sub` in a field, and call
  `<-sub.Stop()` from any method to stop that child. The stopped nested
  supervisor is removed from the supervisor of the run. A child must not
  synchronously call a parent method that stops the group of that child:
  `Stop` waits for the child, and the child waits for the call.
- `proc.StartSupervisor(parent)` nests a supervisor. `Stop` begins stopping;
  its channel closes when all children have stopped, last started first.
  When a nested supervisor's own Stop channel closes, its parent has removed
  its child entry.
  Cancellation is cooperative: `Run` must observe its context. Stop does not
  wait for direct method calls, and methods must synchronize their shared state.
- `proctest.NewRoot(t)` fails a test whose process tree does not stop in 5 s.
