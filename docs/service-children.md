# Child services and groups

Call `p.Children()` inside a service method to start work that stops with that
service run, such as workers in a pool. Each run has its own service value and
child group. A restart creates both again. Choose the parent by the required lifetime:

| Parent | Use it for |
| --- | --- |
| A root from `supervise.NewRoot` | Application work that lasts until application shutdown. |
| A group from `supervise.StartGroup(parent)` | Services that must be stopped together or restarted together after a child exceeds its restart limit. |
| `p.Children()` inside a service method | Work that must stop when that run of `p` ends. |

## Example: build a worker pool in Init

Use Init when all children can be created from the start arguments. The
[worker project](../examples/workers/README.md) creates a
[pool](../examples/workers/pool.go) of queued
[text workers](../examples/workers/worker.go). This part of `Pool.Init` starts
the children and keeps their service references:

```go
children := p.Children()
for id := range p.Args().Workers {
    worker, err := supervise.Start[Worker](children, WorkerArgs{
        ID: id, Version: p.Args().Version,
    })
    if err != nil { return err }
    p.workers = append(p.workers, worker)
}
```

- Child Init runs during Start. Child Run waits until the pool's Run begins.
- Pool.Ready reports when every worker's job loop has started.
- Failed pool Init stops its children. Init must not wait for work from child Run.
- The runtime stops this child group when the parent run ends.
- For children you can stop yourself, create a nested group.
- For work chosen after setup, start children from Run, as the dispatcher does below.
- A restart creates a new parent struct and child group; Init or Run rebuilds the children.

## Example: stop one worker group

Create a nested group under the current run:

```go
group := supervise.StartGroup(p.Children())
worker, err := supervise.Start[Worker](group, WorkerArgs{ID: 1, Version: 1})
if err != nil { panic(err) }
if worker.Current().Ready() {
    result := worker.Current().Process("Hello")
    if result.Available { println(result.Text) } // HELLO
}
```

- StartGroup returns an ordinary `*supervise.Group`; keep it in a field or map.
- `<-group.Stop()` stops its children and removes it from the parent.
- The parent and its other groups continue.
- Release shared-state locks before waiting for Stop. A child may need a parent
  method that takes the same lock to finish shutdown.

## Example: change worker groups from configuration

The [dispatcher](../examples/workers/dispatcher.go) applies this pattern to
configuration changes. Its [configuration service](../examples/workers/config.go)
stores the desired variant names. Each variant gets a group with a worker pool.
The [command](../examples/workers/cmd/workers/main.go) starts two variants:

```go
config, err := supervise.Start[workers.VariantConfig](root, supervise.NoArgs{})
if err != nil { panic(err) }
names := im.FreezeSlice([]string{"blue", "green"})
config.Current().Set(names)
dispatcher, err := supervise.Start[workers.Dispatcher](root, workers.DispatcherArgs{
    Config: config, Workers: 2, Every: time.Millisecond,
})
if err != nil { panic(err) }
dispatcher.Current().Refresh()
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

The dispatcher stages each new version in its own group:

- `Load("blue", 2)` starts a lowercase worker pool for blue.
- Refresh switches pools only after the candidate's Ready method returns true.
- The previous group stops after the switch; green keeps running.
- Failed setup leaves the previous version available.
- Requests for a pending or current version do nothing.

The command waits for each version before it sends a job. Run the whole sequence:

```sh
go run ./examples/workers/cmd/workers
```

On restart, the dispatcher reads the configuration again and creates version-1
pools. This example does not persist version requests. Keep rebuild data in
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
- An old child reference still selects the old child after the parent restarts.
  Keep child references in parent fields and expose methods that select the current child.
  The checker permits Ref copies; it does not prove their lifetime.
- For children that need each other's references, use Reserve and StartReserved in the
  same parent method. Create fresh reservations on each run. Reusing a started
  reservation from the parent's arguments would panic.

Groups follow ordinary Go rules. Exported service methods return no result or
one [service result](service-results.md). Init, Run, and methods supplied by the
runtime are exempt.
