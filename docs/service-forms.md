# Service source forms

These lists cover:

- `supervise.Service` declarations, type names, and receivers.
- Pointers returned by `Ref.Current`.
- `Service.Children` and `supervise.ConcurrentSafe`.

Parentheses preserve an allowed form.

Immutable arguments and results also follow the
[common source forms](common-forms.md). Dependency values use the common
[statement and call forms](common-forms.md#statements-and-calls). Ordinary Go
values outside these rules keep their ordinary Go behavior.

Read [Supervised services](services.md) for a complete FileCache example and
the startup, restart, and stop behavior. The tables below use its FileCache and
FileArgs types. Other examples come from the
[configuration loader](../examples/configuration/config.go),
[limiter](../examples/configuration/limiter.go), and
[worker pool](../examples/workers/pool.go).

## Service declarations

| Allowed forms — exhaustive | Example |
| --- | --- |
| A defined, named, non-generic struct with exactly one `supervise.Service[A]` embedded by value and no im marker | `type Cache struct { supervise.Service[supervise.NoArgs]; mu sync.Mutex; hits int }` |
| `A` is an [allowed stored type](common-forms.md#markers-and-stored-types) | `supervise.Service[int]`, `supervise.Service[FileArgs]` |
| `A` is a `supervise.ConcurrentSafe[T]`, or a struct with one im marker whose fields are allowed stored types or named ConcurrentSafe fields | `supervise.Service[supervise.ConcurrentSafe[*slog.Logger]]`, `supervise.Service[FetcherArgs]` |
| Other service fields use ordinary Go types without protected storage, or the listed [Owner fields](owner.md) | `mu sync.Mutex`, `groups map[string]*supervise.Group`, `loader supervise.Ref[ConfigLoader]`, `current im.Owner[Config]` |
| Declare Run directly on the service with a pointer receiver, one `context.Context` parameter, and no result | `func (c *FileCache) Run(ctx context.Context) { ... }` |
| Declare Init directly with a pointer receiver, no parameters, and exactly one `error` result; or use the embedded default | `func (c *FileCache) Init() error { return c.read() }` |
| Inputs to methods other than Init and Run follow ordinary Go rules | `Each(visit func(string))`, `Values(v ...int)` |
| Declare other methods with a pointer receiver | `func (c *FileCache) read() error { ... }` |
| Exported methods other than Init and Run return only allowed stored types | `func (c *FileCache) Text() string`, `func (l *Limiter) Usage() Usage` |
| Name the concrete service type in its declaration, a pointer receiver, or the single type argument of Start, StartReserved, Reserve, or Ref | `supervise.Start[FileCache](root, args)`, `supervise.Ref[FileCache]` |

- Unexported methods can also return ordinary Go types.
- Protected values inside any method keep their rules.
- Aliases of `context.Context` and `error` retain the required Init or Run signature.

| Rejected examples — non-exhaustive | Allowed replacement |
| --- | --- |
| `type Cache[T any] struct { supervise.Service[T] }`, `var c struct { supervise.Service[supervise.NoArgs] }` | Define a concrete named struct: `type Cache struct { supervise.Service[supervise.NoArgs] }`. |
| `Marker supervise.Service[FileArgs]`, `*supervise.Service[FileArgs]`, or a second marker in the service | Embed one `supervise.Service[FileArgs]` by value. |
| `supervise.Service[map[string]int]`, `supervise.Service[error]` | Use a marked argument struct with allowed fields, such as `supervise.Service[FileArgs]`. |
| A service field `limits im.FrozenMap[string, int]` | Put `Limits im.FrozenMap[string, int]` in the marked arguments, or use `current im.Owner[Config]`. |
| Missing Run, promoted Run, `Run(int)`, or `Run(context.Context) error` | Declare `func (c *FileCache) Run(ctx context.Context)`. |
| `Init(int) error`, `Init()`, `Init() *MyError`, or Init promoted from a user embedded struct | Declare `func (c *FileCache) Init() error`, or use the default from Service. |
| `func (c FileCache) Text() string` | `func (c *FileCache) Text() string` |
| An exported `Table() map[string]int` or `Load() error` | Return `im.FrozenMap[string, int]` or a defined scalar status, respectively. |
| `type Copy = FileCache`, `type Copy FileCache`, `type Outer struct { FileCache }`, or `func use(c *FileCache)` | Use `supervise.Ref[FileCache]` as the field or parameter type. |
| `supervise.Ref[int]`, `supervise.Reserve[Plain]()`, or `func use[T any](r supervise.Ref[T])` | Use a concrete declared service, such as `supervise.Ref[FileCache]`. |
| `supervise.Start[FileCache, FileArgs](root, args)` | Write only the service type argument: `supervise.Start[FileCache](root, args)`. |

## Startup and references

- NewRoot and StartGroup return ordinary `*supervise.Group` values.
- Construct a group before use; its zero value is not running.
- `Children()` has the separate [child-group rules](#child-groups).

| Allowed forms — exhaustive | Example |
| --- | --- |
| Create a root group with an optional failure report | `root := supervise.NewRoot(nil)`, `supervise.NewRoot(func(value any, stack []byte) { log.Printf("%v\n%s", value, stack) })` |
| Start a nested group or request a group's stop | `group := supervise.StartGroup(root)`, `<-group.Stop()` |
| Start a service with its arguments | `cache, err := supervise.Start[FileCache](root, args)` |
| Supply no arguments with NoArgs | `cache, err := supervise.Start[Cache](root, supervise.NoArgs{})` |
| Reserve a concrete reference and initialize it later | `ref := supervise.Reserve[FileCache](); err := supervise.StartReserved(root, ref, args)` |
| Give StartReserved its single explicit service type argument, or infer it from the reference | `err := supervise.StartReserved[FileCache](root, ref, args)` |
| Copy, store, pass, or return a reference | `other := ref`, `type Peers struct { im.Shared; Cache supervise.Ref[FileCache] }`, `func use(ref supervise.Ref[FileCache])` |
| Declare an empty reference for later assignment | `var ref supervise.Ref[FileCache]` |
| Call an exported method declared directly on the service, other than Init or Run, directly and synchronously on Current | `text := ref.Current().Text()`, `(ref.Current()).Text()` |
| Use a reference selected from a marked field | `text := peers.Cache.Current().Text()` |
| Call through a reference inside a function literal | `go func() { ref.Current().Text() }()`, `defer func() { ref.Current().Text() }()` |

- Refs can also be stored in ordinary Go fields and containers.
- Current needs successful initialization; it does not report whether Run is active.
- A zero Ref is empty. Reserve creates storage for StartReserved.

For Start with protected arguments, use these result forms in a common statement
form, including an `if` initializer. Check the returned error in startup code.

| Allowed result forms — exhaustive | Example |
| --- | --- |
| Bind to two local names or blanks with `:=` | `cache, err := supervise.Start[FileCache](root, args)`, `_, err := supervise.Start[FileCache](root, args)` |
| Assign two existing local names or blanks with `=` | `cache, err = supervise.Start[FileCache](root, args)` |
| Declare two local names or blanks with `var` | `var cache, err = supervise.Start[FileCache](root, args)`, `var _, _ = supervise.Start[FileCache](root, args)` |
| Forward the pair from a named function with `(supervise.Ref[FileCache], error)` results | `return supervise.Start[FileCache](root, args)` |
| Discard the results in a direct call statement | `supervise.Start[FileCache](root, args)` |

Start calls whose arguments are ordinary Go values, such as
`supervise.Start[Counter](root, 1)`, use ordinary Go result-binding rules.
StartReserved returns one error and uses the common one-result forms when its
arguments contain protected storage.

| Rejected examples — non-exhaustive | Allowed replacement |
| --- | --- |
| `&FileCache{}`, `new(FileCache)`, `var cache FileCache` | `cache, err := supervise.Start[FileCache](root, args)` |
| `p := ref.Current()`, `saved = ref.Current()`, `use(ref.Current())`, `*ref.Current()`, or `ref.Current() == nil` | Keep the reference; call `ref.Current().Text()` directly. |
| `ref.Current().text`, `ref.Current().Args()`, or a promoted method such as `ref.Current().Lock()` | Declare an exported method on the service, then call it directly: `ref.Current().Text()`. |
| `ref.Current().Init()`, `ref.Current().Run(ctx)` | Only the runtime calls Init and Run. Start the service with `supervise.Start[FileCache](root, args)`. |
| `get := ref.Current`, `read := ref.Current().Text`, `(*FileCache).Text` | `text := ref.Current().Text()` |
| `go ref.Current().Text()`, `defer ref.Current().Text()` | `go func() { ref.Current().Text() }()`, `defer func() { ref.Current().Text() }()` |
| `holder.Ref, err = supervise.Start[FileCache](root, args)` with protected args | Bind local results first, then assign the reference: `ref, err := supervise.Start[FileCache](root, args); holder.Ref = ref`. |

## Inside a service method

These uses apply to the method's receiver, including inside function literals:

| Allowed forms — exhaustive | Example |
| --- | --- |
| Select any receiver field other than the embedded Service | `c.text = text`, `c.mu.Lock()`, `defer c.mu.Unlock()`, `l.current.Store(&Config{})` |
| Call a receiver method directly and synchronously, except Init and Run | `c.read()`, `c.Lock()` when the service embeds `sync.Mutex` |
| Use the same receiver forms inside closures | `func() { c.text = "ready"; c.read() }` |
| Read the start arguments with a direct Args call | `args := c.Args()`, `path := c.Args().Path`, `l.Args().Loader.Current().Limit(l.Args().Key)` |

Selected ordinary fields use ordinary Go rules. Owner fields and any protected
data read from the arguments retain their own source rules.

| Rejected examples — non-exhaustive | Allowed replacement |
| --- | --- |
| `other := c`, `use(c)`, `c == nil`, `*c = FileCache{}` | Select a field or call a method: `c.text`, `c.read()`. |
| `go c.read()`, `defer c.read()`, `read := c.read` | Call `c.read()` directly. Field cleanup such as `defer c.mu.Unlock()` is allowed. |
| `c.Service`, `&c.Service`, `get := c.Args` | Read arguments with `args := c.Args()`. |
| `c.Init()`, `c.Run(ctx)` | Put reusable work in another method, such as `c.read()`. The runtime calls Init and Run. |

## Child groups

Call Children directly on the receiver of a service method, including inside
function literals. Every call in the allowed list is synchronous. For the lifetime of
these groups, see [Child services and groups](service-children.md).

| Allowed forms — exhaustive | Example |
| --- | --- |
| Pass `p.Children()` directly as the first argument of Start, StartReserved, or StartGroup | `supervise.Start[Worker](p.Children(), WorkerArgs{ID: 1, Version: 1})`, `supervise.StartReserved(p.Children(), ref, args)`, `supervise.StartGroup(p.Children())` |
| Bind Children to one new local with `:=`; use that local only as the first argument of those three functions in the same method | `children := p.Children(); supervise.Start[Worker](children, WorkerArgs{ID: 1, Version: 1})` |
| Keep, store, start under, or stop the result of StartGroup; pass it through functions or methods; return it through ordinary functions or unexported methods | `group := supervise.StartGroup(p.Children()); p.groups[name] = group`, later `<-group.Stop()` |

| Group source | Who stops it | Where it can be used |
| --- | --- | --- |
| Children | Runtime | Only the parent positions listed above. |
| StartGroup | Runtime or caller | Ordinary Go uses, but not exported service method results. |

| Rejected examples — non-exhaustive | Allowed replacement |
| --- | --- |
| `p.group = p.Children()`, `saved = p.Children()`, `keep(p.Children())`, `_ = p.Children()` | `supervise.Start[Worker](p.Children(), args)` |
| `p.Children().Stop()`, or `children := p.Children(); <-children.Stop()` | `group := supervise.StartGroup(p.Children())`, then `<-group.Stop()`. |
| `get := p.Children`, `go p.Children()`, `go supervise.Start[Worker](p.Children(), args)`, `defer supervise.StartGroup(children)` | Start the child directly in the method: `supervise.Start[Worker](p.Children(), args)`. |
| `var children = p.Children()`, `children = p.Children()`, `children, n := p.Children(), 1` | `children := p.Children()` |
| After that binding: `other := children`, `keep(children)`, `&children`, `children = root` | Use `children` only as the first argument of Start, StartReserved, or StartGroup. For example, `supervise.StartGroup(children)`. |
| `func (p *Pool) group() *supervise.Group { return p.Children() }` | `return supervise.StartGroup(p.Children())` |
| `ref.Current().Children()`, `p.Service.Children()`, `(*supervise.Service[int]).Children` | Start children from a direct call on the method receiver: `supervise.StartGroup(p.Children())`. |

## Concurrent dependencies

`supervise.ConcurrentSafe[T]` records a caller's promise:

- The value is safe for concurrent use; the checker does not prove this or inspect T.
- Code that creates the value and uses Value's result is still checked.

FetcherArgs is defined in [Concurrent dependencies](services.md#concurrent-dependencies).

| Allowed forms — exhaustive | Example |
| --- | --- |
| Wrap a value with a direct AssumeConcurrentSafe call in any function | `log := supervise.AssumeConcurrentSafe(logger)`, `FetcherArgs{Log: supervise.AssumeConcurrentSafe(logger)}` |
| Declare a named ConcurrentSafe field in a struct with exactly one im marker | `type FetcherArgs struct { im.Shared; Log supervise.ConcurrentSafe[*slog.Logger] }` |
| Use the wrapper itself as the service argument type | `supervise.Service[supervise.ConcurrentSafe[*slog.Logger]]` |
| Use an exact type alias for a wrapper | `type Log = supervise.ConcurrentSafe[*slog.Logger]` |
| Declare a zero wrapper or an empty wrapper literal | `var log supervise.ConcurrentSafe[*slog.Logger]`, `supervise.ConcurrentSafe[*slog.Logger]{}` |
| Use a wrapper or a marked argument struct with dependency fields as a local value, an ordinary function parameter/result, or a start argument | `args := FetcherArgs{Log: log}`, `func makeArgs(log Log) FetcherArgs`, `supervise.Start[Fetcher](root, args)` |
| Call Value directly and synchronously on the method receiver's Args result, or one directly selected field of that result | `f.Args().Log.Value().Info("start")`, `log := f.Args().Log.Value()`, `t.Args().Value()` |

- A zero wrapper's Value returns the zero value of T.
- The returned value, such as `log`, follows ordinary Go rules for its type.
- Binding Args to a local does not permit Value calls through that local.

| Rejected examples — non-exhaustive | Allowed replacement |
| --- | --- |
| A marked start-argument field `Log *slog.Logger` | `Log supervise.ConcurrentSafe[*slog.Logger]`, set with `supervise.AssumeConcurrentSafe(logger)`. |
| A service field `log supervise.ConcurrentSafe[*slog.Logger]`, or that field in an unmarked struct | Keep the wrapper in the marked arguments. An ordinary service field `log *slog.Logger` can hold `f.Args().Log.Value()`. |
| Exported service methods `Log() supervise.ConcurrentSafe[*slog.Logger]` or `Arguments() FetcherArgs` | Return immutable information, for example `Name() string`. |
| `im.Owner[State]` where State has a dependency field, `im.FrozenSlice[Log]`, or `im.FrozenSlice[FetcherArgs]` | Pass FetcherArgs directly to `supervise.Start[Fetcher](root, args)`. |
| A field `Args FetcherArgs`, `Args *FetcherArgs`, or `Logs [2]Log` inside another argument struct | Give the service its direct `supervise.Service[FetcherArgs]` argument type. |
| Embedded `supervise.ConcurrentSafe[*slog.Logger]` in an argument struct | Name the field: `Log supervise.ConcurrentSafe[*slog.Logger]`. |
| `type Log supervise.ConcurrentSafe[*slog.Logger]`, `[]Log`, or `var value any = supervise.AssumeConcurrentSafe(logger)` | Keep the exact wrapper type or its alias: `type Log = supervise.ConcurrentSafe[*slog.Logger]`. |
| A package variable `var saved FetcherArgs` or `var log Log` | Create the arguments in the function that calls Start: `args := FetcherArgs{Log: log}`. |
| `args := f.Args(); args.Log.Value()`, or `func use(log Log) { log.Value() }` | Read through the receiver directly: `f.Args().Log.Value()`. |
| `get := f.Args().Log.Value`, `go f.Args().Log.Value()`, `defer f.Args().Log.Value()` | Read synchronously: `log := f.Args().Log.Value()`. |
