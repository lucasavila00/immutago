# Performance

[Documentation](README.md) · [Collection API](../pkg/im/readme.md)

## Zero abstraction overhead

The target is the C++ zero-overhead principle: equivalent handwritten native Go
should not be faster merely because it avoids `im`. The wrapper should disappear
after inlining. Freeze and reads should add no allocation, element copy, dynamic
dispatch, runtime ownership check, or extra escape to the heap.

Compare the same work. After `frozen := im.FreezeSlice(values)`, the performance
baseline for `frozen.At(i)` is native `values[i]`. A map read has
`value, found := entries[key]` as its baseline. Direct `All()` loops should have
the same overhead as native `range`.

| Operation | Storage work |
| --- | --- |
| `FreezeSlice`, `FreezeMap` | Copy the handle only; no allocation |
| `FrozenSlice.Len`, `FrozenMap.Len` | Constant time; no allocation |
| `FrozenSlice.At` | Native slice indexing; no allocation |
| `FrozenMap.Get` | Native map lookup; no allocation |
| Direct range over `All()` | O(n) traversal; no collection copy |
| `Clone` | O(n) time and independent mutable storage |

There are no persistent update methods. To change a frozen collection, clone it,
edit the clone, and freeze the clone. Existing frozen copies retain the original
storage. Element types must pass the checker so shallow cloning is sufficient.

## Check the cost

[`make bench`](../pkg/im/benchmark_test.go) reports Freeze, read, iteration, and
Clone costs. [Allocation tests](../pkg/im/iterator_test.go) check Freeze, reads,
and direct iteration with prebuilt native storage. Clone tests check that the
copy is independent. These checks cover the current operation costs; they do
not compare every call site with a native baseline.

For a performance change, compare equivalent native and `im` consumer functions
with the same input and observable result. Check inlining, bounds-check removal,
and escape decisions with compiler diagnostics, then inspect generated code
when the results differ. Check both a local-use case and a value returned from
a helper. An allocation test alone cannot show zero abstraction overhead.

Clone explicitly requests an independent copy. Its O(n) work is the cost of
that operation, not an ownership-checking cost. Compare it with the same native
storage copy, not with an in-place update.
