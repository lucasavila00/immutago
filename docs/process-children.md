# Child processes

Use child processes when a service decides which workers must exist. For
example, a dispatcher can keep one worker group for each configuration entry.
When an entry is removed, it stops that group. When the dispatcher stops or
restarts, the runtime stops all of its groups.

If the caller knows the whole tree before startup, it can continue to use
`proc.StartSupervisor(root)` and `proc.Start` directly.

## Start a worker pool

In a process method, pass `p.Supervisor()` to Start. It attaches the child to
that process. The [worker pool](../tests/testdata/accepted/process/worker_pool.go)
starts the number of workers given in its arguments:

```go
func (p *WorkerPool) Run(ctx context.Context) {
    sup := p.Supervisor()
    for i := range p.Args().Workers {
        proc.Start[Worker](sup, WorkerArgs{ID: i})
    }
    <-ctx.Done()
}
```

The runtime stops these workers before it ends the pool's Run context. The
pool does not need a second root or manual child cleanup.

## Add and remove groups while running

The [dispatcher example](../tests/testdata/accepted/process/dispatcher.go)
keeps one group for each configured variant. `Refresh` reads the current
configuration, starts missing groups, and stops removed groups. Each group
contains a throttle and a version loader.

With those example types and an existing root:

```go
config := proc.Start[VariantConfig](root, proc.None{})
initial := im.FreezeSlice([]string{"blue", "green"})
config.Proc().Set(initial)
dispatcher := proc.Start[Dispatcher](root, DispatcherArgs{Config: config})
dispatcher.Proc().Refresh() // start both groups
dispatcher.Proc().Load("blue", 1) // ask blue's loader to start version 1

next := im.FreezeSlice([]string{"green"})
config.Proc().Set(next)
dispatcher.Proc().Refresh() // stop blue and wait for its children
```

Inside `Refresh`, a new group starts with
`sub := proc.StartSupervisor(d.Supervisor())`. The dispatcher starts its workers
under `sub` and keeps `sub` in a field. To remove the group, it removes the field
entry, releases its mutex, and waits on `<-sub.Stop()`. Stop also removes the
group from its parent, so repeated additions and removals do not leave old
groups in the tree.

The version loader uses the same pattern for each loaded version. The new
server calls `Ready` when it starts; only then does the loader stop the previous
server. A request to Load does not mean the version is ready. Repeated requests
for a pending or current version do nothing.

`Dispatcher.Run` calls `Refresh` too. After a restart, the new dispatcher has
empty fields, so it rebuilds its groups from the config process. Keep the data
needed to rebuild children in start arguments or in a peer process. Mutable
fields of the old dispatcher do not survive its restart.

## Stop and failure

| Event | Result |
| --- | --- |
| A method starts a child before its parent's Run begins | The child waits for that run to start. If the parent never runs, neither does the child. |
| The parent is stopped | Children stop last started first. The runtime waits for them, then ends the parent's Run context. Children can call the parent while they stop. |
| The parent's Run returns or panics | The runtime stops and waits for its children before the run ends or the restart delay begins. |
| A child passes its restart limit | Its failure ends the parent run. A nested group first restarts by itself; repeated group failures reach the parent. The normal restart limits apply. |
| A method starts a child after that parent run ends | Start returns a PID, but the child does not run. The same applies to a nested supervisor kept from that run. |

Only the runtime stops `p.Supervisor()`. To stop selected children from a
method, use a nested supervisor as above. A child must not wait for a parent
method that stops the child's own group: Stop would wait for the same child.
As with other processes, Run must observe cancellation, and Stop does not wait
for direct method calls.

## Child lifetimes

A child PID does not follow a restart of its parent. It still points to the
old child, whose Run has ended. Keep child PIDs in the parent and expose parent
methods that select the current child. The checker permits PID copies; it does
not prove that a PID belongs to the current run.

For children that need reserved PIDs, call Reserve in the same method as
StartReserved. Do not put a reserved child PID in the parent's start arguments:
the next run would try to start that PID again and panic with
`proc: PID already started`.

The [process reference](process-forms.md#start-children) lists all allowed
Supervisor calls and nearby rejected forms.
