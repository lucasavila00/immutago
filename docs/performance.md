# Collection costs

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
A focused 200-reference CLI comparison reduced peak RSS from 348.2 MiB to
97.8 MiB. This includes package loading and is one measurement, not a general
memory bound. See [the measured review comparison](arch/validation.md).
