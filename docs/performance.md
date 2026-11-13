# Collection costs

[Project reference](README.md)

The checker adds no runtime proof checks. Frozen collections wrap native Go
storage. Freeze adds no allocation and does not copy the input.

| Operation | Storage work |
| --- | --- |
| `FreezeSlice`, `FreezeMap` | Copy the handle only; no allocation |
| `FrozenSlice.Len`, `FrozenMap.Len` | Constant time; no allocation |
| `FrozenSlice.At` | Native slice indexing; no allocation |
| `FrozenMap.Get` | Native map lookup; no allocation |
| Direct range over `All()` | O(n) traversal; no collection copy |
| `Clone` | O(n) time and independent mutable storage |

Native construction still costs memory. A preallocated slice needs its backing
array; a Go map can use several internal allocations depending on size and Go
version. We promise no second collection allocation at Freeze, not exactly one
allocation for all construction.

There are no persistent update methods. To change a frozen collection, clone it,
edit the clone, and freeze the clone. Existing frozen copies retain the original
storage. Element types must pass the checker so shallow cloning is sufficient.

`make bench` reports Freeze, read, iteration, and clone costs. Allocation tests
check Freeze and reads separately from native construction. Clone tests check
that changes to the clone do not change the frozen source.

## Checker memory

Fixed references have one owner and need no forward possible-owner maps.
The earlier solver retained pointer-to-owner maps at every operation point,
including dead bindings. The current solver stores one owner per local
reference and retains backward liveness.

A focused same-machine comparison used separate blocks in one function:

```go
{ var v Value; p := &v; _ = p.X; v.X = 1 }
```

| Blocks | Before: peak RSS (MiB) | Fixed owners: peak RSS (MiB) |
| --- | ---: | ---: |
| 50 | 107.5 | 104.0 |
| 100 | 157.0 | 100.2 |
| 200 | 348.2 | 97.8 |

The baseline was `269eaaf`; the changed solver was `8753467`. Both binaries
checked the same generated consumer packages with
`/usr/bin/time -l <binary> ./scale<N>`, where N was 50, 100, or 200. All six checks passed.
These single observations include package loading and are not isolated solver
benchmarks. At 200 blocks, elapsed time was 0.44 seconds before and 0.19 seconds
after. The memory growth from retained possible-owner maps is removed; these
measurements do not establish a bound for all source programs.
