# Child services and groups

A service can own work that must stop with its current run. For example, a
worker pool owns its workers, and a dispatcher owns one group for each active
configuration entry. Call `Children()` inside a service method to attach that
work to the service.

Choose the parent that matches the required lifetime:

| Parent | Use it for |
| --- | --- |
| A root from `supervise.NewRoot` | Application work that lasts until application shutdown. |
| A group from `supervise.StartGroup(parent)` | Services that must be stopped together or restarted together after a child exceeds its restart limit. |
| `p.Children()` inside a service method | Work that must stop when that run of `p` ends. |

## Build a pool during initialization

Use Init when all children can be created from the start arguments. This
[pool](../tests/testdata/accepted/service/pool_init.go) uses Worker and WorkerArgs from the
[worker example](../tests/testdata/accepted/service/worker_pool.go):

```go
// Pool owns the requested number of workers.
type Pool struct {
    supervise.Service[int]
}

// Init creates the workers before the pool becomes available.
func (p *Pool) Init() error {
    children := p.Children()
    for id := range p.Args() {
        if _, err := supervise.Start[Worker](children, WorkerArgs{ID: id}); err != nil {
            return err
        }
    }
    return nil
}

// Run keeps the pool active until shutdown.
func (*Pool) Run(ctx context.Context) { <-ctx.Done() }
```

Each child completes its Init when Start is called. Its Run waits until the
pool's Run begins. If the pool's Init fails, the runtime stops the children
created during that attempt. Init must not wait for work that a child's Run
will produce.

`children := p.Children()` is a local way to reuse the parent argument. In that
method, use `children` only as the first argument of Start, StartReserved, or
StartGroup. The runtime owns this group and stops it. To stop selected children
yourself, put them in a nested group.

The worker example also creates children from Run. Use that form when the work
is chosen after initialization. In both cases, a restart creates a new parent
struct and a new child group. Init or Run creates the children again.

## Stop one group while the parent continues

Create a nested group under the current run:

```go
group := supervise.StartGroup(p.Children())
worker, err := supervise.Start[Worker](group, WorkerArgs{ID: 1})
if err != nil { panic(err) }
worker.Current().Do()
```

The result of StartGroup is an ordinary `*supervise.Group`. A service can keep
it in a field or a map. Later, `<-group.Stop()` stops its children and removes
the group from its parent. The parent service and its other groups continue.

When a method removes a group from shared state, release its mutex before
waiting for Stop. A child can need a parent method while it stops. Holding a
mutex that the method needs would prevent shutdown.

The [dispatcher](../tests/testdata/accepted/service/dispatcher.go) applies this
pattern to configuration changes. Its configuration service stores the desired
variant names. Each variant gets a group with a throttle and a version loader.
With the example types and an existing root:

```go
config, err := supervise.Start[VariantConfig](root, supervise.NoArgs{})
if err != nil { panic(err) }
names := im.FreezeSlice([]string{"blue", "green"})
config.Current().Set(names)
dispatcher, err := supervise.Start[Dispatcher](root, DispatcherArgs{Config: config})
if err != nil { panic(err) }
dispatcher.Current().Refresh()
dispatcher.Current().Load("blue", 1)
```

Refresh compares the desired names with its current groups. It calls
`supervise.StartGroup(d.Children())` for each new name. For a removed name, it
deletes the saved group, releases its mutex, and waits for that group to stop.
For example, this change removes blue and keeps green:

```go
names = im.FreezeSlice([]string{"green"})
config.Current().Set(names)
dispatcher.Current().Refresh()
```

The version loader uses a separate group for each loaded version. It stops the
previous group only after the new server calls Ready. A Load call starts the
work; it does not mean that the version is ready. Repeated requests for a
pending or current version do nothing in this example.

The dispatcher reads the configuration again when its Run restarts. Keep the
data needed to rebuild children in start arguments or in a service that outlives
the parent. Mutable fields of the old parent are not copied to its replacement.

## Child lifetimes

| Event | Effect on the children |
| --- | --- |
| The parent's Init returns an error or panics | Children from that setup attempt stop. The runtime does not close files or other application resources opened by Init; Init must clean them up on failure. |
| A method starts a child before the parent's Run begins | The child finishes Init, then waits to run. If the parent never runs, the child never runs. |
| The parent is stopped | Children stop last started first. The runtime waits for them before it ends the parent's Run context. |
| The parent's Run returns or panics | The runtime stops and waits for its children before the run ends or the restart delay begins. |
| A child exceeds its restart limit | The containing group stops its children and passes the failure up. A nested group can restart first; a failure that reaches the service ends that service's run. |
| A method starts a child after the parent run has ended | Init still runs. If it succeeds, Start returns a reference, but the child's Run does not run. A nested group saved from that run has the same limit. |

Children can call their parent while they stop. They must not wait for a parent
method that stops their own group: that Stop would wait for the same child.
As with any service, Run must return when its context ends. Stop waits for Run,
not for direct method calls.

A child reference belongs to the run that created the child. If the parent
restarts, an old child reference still points to the old child. It does not
switch to a child created by the new parent. Keep child references in the
parent's fields and expose parent methods that select the current child.
The checker permits reference copies; it does not prove this lifetime.

If children need references to each other, call Reserve and StartReserved in
the same parent method. Create new reservations each time that method builds
the children. A reserved child reference in the parent's start arguments would
be reused on the next run; starting that reference again would panic.

See [child-group forms](service-forms.md#child-groups) for the complete allowed
calls and nearby rejected forms.
