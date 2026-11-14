# Process reference

For a complete example and the execution, restart, and stop rules, read
[Supervised processes](processes.md).

The [config loader](../tests/testdata/accepted/process/config_loader.go) and
[limiter](../tests/testdata/accepted/process/limiter.go) examples show Owner fields
and PIDs in start arguments. The
[worker pool](../tests/testdata/accepted/process/worker_pool.go) starts children;
the [dispatcher](../tests/testdata/accepted/process/dispatcher.go) adds and removes
child groups from its methods. The tables also use `FetcherArgs` from
[unchecked dependencies](processes.md#unchecked-dependencies).

These allowed forms are complete. Other uses of a process, its type name, a
process pointer from `Proc()`, or a `proc.UncheckedDependency` are rejected. Parentheses preserve these forms.
Dependency values also use the [common statement forms](common-forms.md#statements-and-calls).

## Declare a process

| Allowed form | Example |
| --- | --- |
| Named, non-generic struct with one `proc.Process[A]` embedded by value; `A` is an [allowed stored type](common-forms.md#markers-and-stored-types), a `proc.UncheckedDependency[T]`, or a struct with exactly one `im` marker whose fields are allowed stored types or named `proc.UncheckedDependency[T]` fields; the other fields are ordinary Go without [protected storage](rules.md), or [Owner](owner.md) fields | `type ConfigLoader struct { proc.Process[LoaderArgs]; mu sync.Mutex; current im.Owner[Config] }` |
| Process without arguments | `type Cache struct { proc.Process[proc.None]; hits int }` |
| Declare the entry point on the process | `func (l *ConfigLoader) Run(ctx context.Context) { ... }` (no results) |
| Method with a pointer receiver | `func (l *ConfigLoader) Version() int { return l.version }` |
| Exported method other than `Run` whose parameters and results are allowed stored types | `func (l *ConfigLoader) Limit(key string) int`, `func (l *Limiter) Usage() Usage` |

## Use fields and arguments inside methods

| Allowed form | Example |
| --- | --- |
| In a method, select a field of the receiver other than the marker | `l.version++`, `l.mu.Lock()`, `defer l.mu.Unlock()`, `l.current.Store(&Config{})` |
| In a method, call a method of the receiver directly and synchronously, other than `Run` | `l.publish()`, `l.Lock()` for an embedded `sync.Mutex` |
| In a method, read the arguments with a direct call | `args := l.Args()`, `l.Args().Loader.Proc().Limit(l.Args().Key)` |

## Call methods and pass PIDs

| Allowed form | Example |
| --- | --- |
| Call an exported method other than `Run`, declared on the process, directly and synchronously on `pid.Proc()` | `v := loader.Proc().Version()`, `(loader.Proc()).Version()` |
| Call through a PID in a field of a marked value | `peers.Loader.Proc().Version()` |
| Keep, copy, store, pass, or return a PID | `type Peers struct { im.Shared; Loader proc.PID[ConfigLoader] }` |
| Call through a PID in a function literal | `defer func() { limiter.Proc().Take() }()` |

## Start processes

| Allowed form | Example |
| --- | --- |
| Start a process with its arguments | `loader := proc.Start[ConfigLoader](root, LoaderArgs{Path: "config.json", Every: time.Minute})` |
| Reserve a PID and start it later, for processes that need each other's PID | `pong := proc.Reserve[Pong](); ping := proc.Start[Ping](root, pong); proc.StartReserved(root, pong, ping)` |
| Name a process type in its declaration, a pointer receiver, or the single type argument of `proc.Start`, `proc.StartReserved`, `proc.Reserve`, or `proc.PID` | `proc.PID[ConfigLoader]`, `proc.Reserve[Pong]()` |

## Start children under the supervisor of the run

| Allowed form | Example |
| --- | --- |
| In a method, call `Supervisor()` directly and synchronously on the receiver, and pass the result directly as the first argument of a direct, synchronous `proc.Start`, `proc.StartReserved`, or `proc.StartSupervisor` call | `proc.Start[Worker](p.Supervisor(), WorkerArgs{ID: i})`, `proc.StartReserved(p.Supervisor(), spare, args)`, `proc.StartSupervisor(p.Supervisor())` |
| In a method, bind `Supervisor()` to one new local variable with `:=`, and use that variable only as the first argument of a direct, synchronous `proc.Start`, `proc.StartReserved`, or `proc.StartSupervisor` call in the same method, not in a function literal | `sup := p.Supervisor(); for i := range p.Args().Workers { proc.Start[Worker](sup, WorkerArgs{ID: i}) }` |
| Keep, store, start under, and stop the result of `proc.StartSupervisor(p.Supervisor())`; it is an ordinary `*proc.Supervisor` | `sub := proc.StartSupervisor(d.Supervisor()); d.groups[name] = group{sub: sub}`, later `<-g.sub.Stop()` |

## Unchecked dependencies

| Allowed form | Example |
| --- | --- |
| Call `proc.Trust` directly to wrap a value in any function; use the result in the value forms below | `log := proc.Trust(logger)`, `FetcherArgs{Log: proc.Trust(logger)}` |
| Declare a named `proc.UncheckedDependency[T]` field in a struct with one `im` marker; the checker does not inspect `T` through the wrapper | `type FetcherArgs struct { im.Shared; Log proc.UncheckedDependency[*slog.Logger] }` |
| Use an alias for a dependency type | `type Log = proc.UncheckedDependency[*slog.Logger]` |
| Declare a zero dependency or an empty dependency literal; `Value` returns the zero value of `T` | `var log proc.UncheckedDependency[*slog.Logger]`, `proc.UncheckedDependency[*slog.Logger]{}` |
| Use a `proc.UncheckedDependency[T]` or a struct with a dependency field as a local value, an ordinary function parameter or result, or a start argument | `args := FetcherArgs{Log: log}`, `proc.Start[Fetcher](root, args)`, `proc.Start[Tracer](root, proc.Trust(logger))` |
| In a method, call `Value` directly and synchronously on `l.Args()` or on one field of `l.Args()`, where `l` is the receiver | `f.Args().Log.Value().Info("start")`, `log := f.Args().Log.Value()`, `t.Args().Value()` for `proc.Process[proc.UncheckedDependency[*slog.Logger]]` |

## Rejected examples

These examples are not exhaustive.

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
| `go loader.Proc().Version()` | `go func() { loader.Proc().Version() }()` |
| `defer loader.Proc().Version()` | `defer func() { loader.Proc().Version() }()` |
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
| `type Log proc.UncheckedDependency[*slog.Logger]`, `[]proc.UncheckedDependency[*slog.Logger]`, `var x any = proc.Trust(logger)` | Keep the dependency value in its exact type or an alias, then pass it in the start arguments. |
| A package variable of dependency type, or of a struct with a dependency field | Build the arguments in the function that starts the process. |
| `args := f.Args(); args.Log.Value()`, `func use(d proc.UncheckedDependency[*slog.Logger]) { d.Value() }`, `get := f.Args().Log.Value`, `go f.Args().Log.Value()` | `f.Args().Log.Value()` in a process method. |
