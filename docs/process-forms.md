# Process reference

For a complete example and the execution, restart, and stop rules, read
[Supervised processes](processes.md).

The [config loader](../tests/testdata/accepted/process/config_loader.go) and
[limiter](../tests/testdata/accepted/process/limiter.go) examples show Owner fields
and PIDs in start arguments. The tables use those example types.

These allowed forms are complete. Other uses of a process, its type name, or a
process pointer from `Proc()` are rejected. Parentheses preserve these forms.

## Declare a process

| Allowed form | Example |
| --- | --- |
| Named, non-generic struct with one `proc.Process[A]` embedded by value; `A` is an [allowed stored type](common-forms.md#markers-and-stored-types); the other fields are ordinary Go without [protected storage](rules.md), or [Owner](owner.md) fields | `type ConfigLoader struct { proc.Process[LoaderArgs]; mu sync.Mutex; current im.Owner[Config] }` |
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
| Use a constant | `const maxStep = 3`; `return min(i, maxStep)` |
| Call or name a function or method whose body, and the bodies of its static callees in checked packages, use only the variables above | `func publishRates(base int) { rates.Store(&Rates{Base: base}) }`; `publishRates(r.base)` |

A variable is changed outside its declaration if code outside `init` functions
assigns it or one of its elements or fields, uses `++` or `--` on it, assigns it
in a `for ... = range` statement, takes its address with `&`, slices it when it
is an array, or calls a pointer method on it. These are the Go operations that
change or alias a variable.

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
| `os.Stdout`, `os.Args`, `http.DefaultClient`, `io.EOF` | No allowed form yet for these dependencies. [Issue #52](https://github.com/lucasavila00/immutago/issues/52) tracks explicit start arguments for them. |
| `var errMissing error = code("missing")` | `const errMissing code = "missing"`, where `code` is a defined string type with an `Error` method. |
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
| `&ConfigLoader{}`, `var l ConfigLoader`, `new(ConfigLoader)` | `proc.Start[ConfigLoader](root, args)` |
| `func use(l *ConfigLoader)`, `type L = ConfigLoader`, `type Outer struct { ConfigLoader }`, `(*ConfigLoader).Version` | Pass a `proc.PID[ConfigLoader]`. |
| `proc.Start[ConfigLoader, LoaderArgs](root, args)` | `proc.Start[ConfigLoader](root, args)` |
| `proc.PID[int]`, `proc.Reserve[Plain]()`, `func f[T any](p proc.PID[T])` | Use a concrete process type. |
| `type Bad struct { proc.Process[map[string]int] }`, `proc.Process[error]` | Use an allowed argument type, for example `proc.Process[LoaderArgs]`. |
| `type Bad[T any] struct { proc.Process[T] }`, `var x struct{ proc.Process[proc.None] }` | Declare a named, non-generic struct. |
| `type Bad struct { proc.Process[proc.None]; im.Shared }`, `M proc.Process[proc.None]`, `*proc.Process[proc.None]` | Embed one `proc.Process[A]` by value and no `im` marker. |
| `type Bad struct { proc.Process[proc.None]; limits im.FrozenMap[string, int] }` | Put immutable data in the arguments, or publish it through an `im.Owner` field. |
