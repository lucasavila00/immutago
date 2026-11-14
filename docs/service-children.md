# Child services and groups

Call `p.Children()` inside a service method to start work that stops with that
run, such as workers in a pool. Choose the parent by the required lifetime:

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

- Child Init runs during Start. Child Run waits until the pool's Run begins.
- Failed pool Init stops its children. Init must not wait for work from child Run.
- Use `children` only as the parent argument of Start, StartReserved, or StartGroup
  in that method. The runtime stops this group.
- For children you can stop yourself, create a nested group.
- For work chosen after setup, start children from Run, as in the worker example.
- A restart creates a new parent struct and child group; Init or Run rebuilds the children.

## Stop one group while the parent continues

Create a nested group under the current run:

```go
group := supervise.StartGroup(p.Children())
worker, err := supervise.Start[Worker](group, WorkerArgs{ID: 1})
if err != nil { panic(err) }
worker.Current().Do()
```

- StartGroup returns an ordinary `*supervise.Group`; keep it in a field or map.
- `<-group.Stop()` stops its children and removes it from the parent.
- The parent and its other groups continue.
- Release shared-state locks before waiting for Stop. A child may need a parent
  method that takes the same lock to finish shutdown.

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

Refresh compares desired names with current groups:

- New name: call `supervise.StartGroup(d.Children())`.
- Removed name: delete the saved group, release the mutex, then wait for Stop.

This change removes blue and keeps green:

```go
names = im.FreezeSlice([]string{"green"})
config.Current().Set(names)
dispatcher.Current().Refresh()
```

The version loader uses one group per version:

- Load starts the work; Ready marks the new server as ready.
- The previous group stops only after Ready.
- Requests for a pending or current version do nothing.

On restart, the dispatcher reads the configuration again. Keep rebuild data in
start arguments or a longer-lived service; old mutable fields are not copied.

## Child lifetimes

| Event | Effect on the children |
| --- | --- |
| The parent's Init returns an error or panics | Children from that setup attempt stop. The runtime does not close files or other application resources opened by Init; Init must clean them up on failure. |
| A method starts a child before the parent's Run begins | The child finishes Init, then waits to run. If the parent never runs, the child never runs. |
| The parent is stopped | Children stop last started first. The runtime waits for them before it ends the parent's Run context. |
| The parent's Run returns or panics | The runtime stops and waits for its children before the run ends or the restart delay begins. |
| A child exceeds its restart limit | The containing group stops its children and passes the failure up. A nested group can restart first; a failure that reaches the service ends that service's run. |
| A method starts a child after the parent run has ended | Init still runs. If it succeeds, Start returns a reference, but the child's Run does not run. A nested group saved from that run has the same limit. |

- Children can call the parent during shutdown, but must not wait for a parent
  method that stops their own group: that Stop waits for the same child.
- Run must return when its context ends. Stop does not wait for direct calls.
- An old child Ref still selects the old child after the parent restarts.
  Keep child Refs in parent fields and expose methods that select the current child.
  The checker permits Ref copies; it does not prove their lifetime.
- For children that need each other's Refs, use Reserve and StartReserved in the
  same parent method. Create fresh reservations on each run. Reusing a started
  reservation from the parent's arguments would panic.

See [child-group forms](service-forms.md#child-groups) for the complete allowed
calls and nearby rejected forms.
