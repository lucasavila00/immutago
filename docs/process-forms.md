# Process reference

For a complete example and the execution, restart, and stop rules, read
[Supervised processes](processes.md).

The [config loader](../tests/testdata/accepted/process/config_loader.go) and
[limiter](../tests/testdata/accepted/process/limiter.go) examples show Owner fields
and PIDs in start arguments. The
[worker pool](../tests/testdata/accepted/process/worker_pool.go) example starts
children under the supervisor of its run. The
[dispatcher](../tests/testdata/accepted/process/dispatcher.go) example starts
and stops groups of children from its methods during one run. The tables use
those example types, and the
`FetcherArgs` type from [unchecked dependencies](processes.md#unchecked-dependencies).

These allowed forms are complete. Other uses of a process, its type name, a
process pointer from `Proc()`, or a `proc.UncheckedDependency` are rejected. Parentheses preserve these forms.

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
| Wrap a value in a dependency, anywhere | `log := proc.Trust(logger)`, `FetcherArgs{Log: proc.Trust(logger)}` |
| Declare a named `proc.UncheckedDependency[T]` field in a struct with one `im` marker; `T` has no protected storage | `type FetcherArgs struct { im.Shared; Log proc.UncheckedDependency[*slog.Logger] }` |
| Use a `proc.UncheckedDependency[T]` or a struct with a dependency field as a local value, an ordinary function parameter or result, or a start argument | `args := FetcherArgs{Log: log}`, `proc.Start[Fetcher](root, args)`, `proc.Start[Tracer](root, proc.Trust(logger))` |
| In a method, call `Value` directly and synchronously on `l.Args()` or on one field of `l.Args()`, where `l` is the receiver | `f.Args().Log.Value().Info("start")`, `log := f.Args().Log.Value()`, `t.Args().Value()` for `proc.Process[proc.UncheckedDependency[*slog.Logger]]` |
## Package variables

A process method (any method with a process receiver, including `Run`) cannot
use a package variable that another goroutine can change. This rule also
applies to the functions that the method calls. The checker accepts a direct
use or a call only in the forms below. The list is complete; it rejects all
other uses of package variables in process methods.

| Allowed form | Example |
| --- | --- |
| Use a package `im.Owner` variable of any package with the [Owner forms](owner.md) | `var rates im.Owner[Rates]`; `rates.Store(&Rates{Base: 1})`, `r := rates.Load()` |
| Read an unexported variable of the same package whose type is an allowed stored type, when only its declaration and `init` functions change it | `var defaultBase = 10`; `return r.base * defaultBase` |
| Call a method directly on an unexported `atomic.Bool`, `atomic.Int32`, `atomic.Int64`, `atomic.Uint32`, or `atomic.Uint64` variable of the same package, when every use of the variable in the package is the receiver of a direct method call | `var lookups atomic.Int64`; `lookups.Add(1)`, `n := lookups.Load()`, `lookups.CompareAndSwap(1, 2)` |
| Compare with an `error` variable of any package: an operand of `==` or `!=`, a value of a `switch` case, or the second argument of `errors.Is`, when no checked code changes the variable after package initialization | `err == io.EOF`, `errors.Is(err, state.ErrFull)`, `switch err { case io.EOF: }` |
| Use a variable of any package whose type is a struct with only empty struct fields | `binary.LittleEndian.PutUint64(out[:], v)`, `body == http.NoBody` |
| Use a constant | `const maxStep = 3`; `return min(i, maxStep)` |
| Call or name a function or method whose body, and the bodies of its static callees in checked packages, use only the variables above | `func publishRates(base int) { rates.Store(&Rates{Base: base}) }`; `publishRates(r.base)` |

A variable is changed outside its declaration if code outside `init` functions
assigns it or one of its elements or fields, uses `++` or `--` on it, assigns it
in a `for ... = range` statement, takes its address with `&`, slices it when it
is an array, or calls a pointer method on it. These are the Go operations that
change or alias a variable.

Checked code cannot assign an `error` variable of another package or take its
address: `io.EOF = nil` and `&state.ErrFull` are rejected in every checked
package. An exported `error` variable that its own package changes outside
`init` carries a fact, so importers do not accept it in a comparison.

The checker looks at a function, or a concrete method, that the process method
names directly: a call, `go` or `defer`, or a function value such as
`f := record`. Each function of a checked package carries one fact: whether its
body, or a function it names, uses a rejected variable. Code in a function
literal counts as code of the function that contains it.

Rejected examples are not exhaustive:

| Rejected example | Allowed replacement |
| --- | --- |
| `var seen = map[string]int{}`; `seen["hits"]++` in `Run` | Keep the map in a process field and give the other processes an exported method. |
| `var seen = map[string]int{}`; `func record(k string) { seen[k]++ }`; `record("hits")` in `Run` | Keep the map in a process field; call `record` from that process only, with the map as a parameter. |
| `func outer() { record("a") }`; `outer()` in a method | The same: the checker follows the chain of static calls. |
| `var mu sync.Mutex` and `var table map[string]int`, used under the lock | Keep both in one process. |
| `var Limit = 10` (exported), `state.Limit` from another package | Pass the value in the start arguments: `proc.Start[RateKeeper](root, Args{Limit: 10})`. |
| `var retries = 3` with `func set(n int) { retries = n }` | Pass the value in the start arguments, or publish it through an `im.Owner` field. |
| `os.Stdout`, `os.Stderr`, `http.DefaultClient`, `http.DefaultTransport`, a package logger, `time.UTC`, `base64.StdEncoding`, a package `*regexp.Regexp`, `sync.Pool`, `sync.Once`, a package template, `embed.FS`, a metric vector | Give the process the dependency in its start arguments with `proc.UncheckedDependency` ([#60](https://github.com/lucasavila00/immutago/pull/60)), or keep a read-only structure in a marked object with `im.Opaque` ([#61](https://github.com/lucasavila00/immutago/pull/61)). |
| `return io.EOF`, `io.EOF.Error()`, `fmt.Errorf("read: %w", io.EOF)`, `errors.As(err, &target)` with a package variable | Compare the error in the process method, then return a defined status or a constant error. |
| `var lookups atomic.Int64` with `p := &lookups` or `f := lookups.Add` anywhere in the package | Call the methods directly: `lookups.Add(1)`. |
| `var current atomic.Value`, `atomic.Pointer[T]`, `atomic.Uintptr`, or a struct or array of atomics | Publish the value through an `im.Owner` field. |
| `p := &defaultBase` in a method | `v := defaultBase` |

Diagnostics name the variable, or the function and the variable it reaches:

```text
process method uses package variable seen, whose type map[string]int is not an allowed stored type; ...
process method calls record, which uses package variable seen; ...
```

## Calls that are not followed

The checker does not follow two kinds of calls:

- A call into a package that the checker does not check (see the
  [package boundary](rules.md#package-boundary)): the standard library,
  third-party packages, and your packages that do not use `im` or `proc`.
  `slog.Info("started")` and `log.Printf(...)` pass, and so does
  `plain.Remember("a")` where `plain` has no `im` or `proc` use and
  `func Remember(k string) { names[k] = k }` changes a package map. A direct
  use of a variable of these packages, such as `os.Stdout`, is still rejected.
  So `slog.Info("started")` passes because it is a function call, while a
  package variable `var log = slog.Default()` is rejected because a process
  method uses the variable directly.
- A call through an interface method. The concrete method is not known:

  ```go
  type recorder interface{ record(string) }
  type mapRecorder struct{}
  func (mapRecorder) record(k string) { seen[k]++ }

  func (w *Worker) Run(ctx context.Context) {
      var r recorder = mapRecorder{}
      r.record("hits") // passes: the checker does not know which record runs
  }
  ```

  A function value is followed where checked code names the function or
  contains the function literal. A function value that comes from an interface
  method or an unchecked package is not followed.

Code in unchecked packages can still assign an `error` variable of another
package, for example `io.EOF = nil` in a third-party package. The checker
cannot see that assignment, so a comparison with `io.EOF` stays allowed.

Package initialization is trusted: do not start a process from a package
variable initializer or an `init` function, because those can still change the
variables that the process reads.

These limits are not a way to share state. `go test -race` finds a race through
them; the checker does not.

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
| `proc.Trust(&Snapshot{})`, where `Snapshot` embeds `im.Shared` | Put the immutable value in a field of the arguments. |
| `args := f.Args(); args.Log.Value()`, `func use(d proc.UncheckedDependency[*slog.Logger]) { d.Value() }`, `get := f.Args().Log.Value`, `go f.Args().Log.Value()` | `f.Args().Log.Value()` in a process method. |
