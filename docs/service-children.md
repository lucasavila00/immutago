# Child services and groups

Call `p.Children()` inside a service method to start work that stops with that
service run, such as workers in a pool. Each run has its own service value and
child group. A restart creates both again. Choose the parent by the required lifetime:

| Parent | Use it for |
| --- | --- |
| A root from `supervise.NewRoot` | Application work that lasts until application shutdown. |
| A group from `supervise.StartGroup(parent)` | Services that must be stopped together or restarted together after a child exceeds its restart limit. |
| `p.Children()` inside a service method | Work that must stop when that run of `p` ends. |

## Example: stop a document index

A nested group lets an application stop an index while other root services
continue. This uses the [search application's indexer](../examples/search/indexer.go):

```go
group := supervise.StartGroup(root)
indexer, err := supervise.Start[search.Indexer](group, search.IndexerArgs{
    Files: os.DirFS("examples/search/documents"), Every: time.Second,
})
if err != nil {
    <-group.Stop()
    panic(err)
}
handler := search.HTTPHandler(indexer)
```

Use handler for the index's HTTP requests. To remove that work, stop accepting
new requests and wait for active handlers to finish. Then stop the group:

```go
<-group.Stop()
```

- Use `p.Children()` instead of root when the group belongs to one run of a parent service.
- Child Init runs during Start. A child created during parent Init waits for the
  parent's Run before its own Run can begin. Init must not wait for child Run.
- A parent restart creates a new child group. Create its children again from
  Init or Run; do not reuse the previous run's group.
- Stop removes a nested group from its parent. Other groups continue.
- Release shared-state locks before waiting for Stop; a child may need those
  locks to finish.

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
