# Frozen collections

Use `im.FrozenSlice[T]` or `im.FrozenMap[K, V]` to build data once and pass it to
readers without letting them change it. Prepare a normal Go slice or map, then
call Freeze. Freeze reuses that storage; it does not copy the entries.

To make a changed version, Clone the frozen collection, edit the copy, and
freeze it. The original stays unchanged:

```go
package main

import "github.com/lucasavila00/immutago/pkg/im"

func main() {
    scores := map[string]int{"alice": 10}
    frozen := im.FreezeMap(scores)

    next := frozen.Clone()
    next["alice"] = 20
    updated := im.FreezeMap(next)

    oldScore, _ := frozen.Get("alice")
    newScore, _ := updated.Get("alice")
    println(oldScore, newScore) // 10 20
}
```

After Freeze, read through `frozen`, not `scores`. The checker rejects aliases
to the original slice or map, so another variable cannot change the frozen data.
Keys and elements must use the [allowed types](common-forms.md#markers-and-stored-types).

The tables below and the [common forms](common-forms.md) give the complete list
of allowed operations. Rejected examples are not exhaustive.

## Build before freezing

A **builder** is the Go slice or map that you prepare before Freeze. These rules
also apply to Go containers whose keys or elements contain protected storage.
Keep the slice or map in one local variable until Freeze. Clone copies the
slice or map; pointer elements still point to the same read-only objects.

| Allowed form | Example |
| --- | --- |
| Declare from make, literal, zero, or Clone | `values := make([]int, 2)`, `values := []int{1}`, `var values []int`, `values := frozen.Clone()` |
| Initialize with bare `nil` | `var values []int = nil`, `var scores map[string]int = nil` |
| Read or replace an element | `x := values[i]`, `values[i] = x`, `scores[key] = x` |
| Read length or slice capacity | `len(values)`, `cap(values)` |
| Append individual elements to the same variable | `values = append(values, 1, 2)` |
| Delete, clear, or copy | `delete(scores, key)`, `clear(values)`, `copy(destination, source)` |
| Range over elements | `for i, value := range values { ... }` |
| Freeze the slice or map | `frozen := im.FreezeSlice(values)`, `frozen := im.FreezeMap(scores)` |

`var` declarations also work. `copy` operands must satisfy the builder rules.
Bare `nil` means Go's predeclared `nil`; `(nil)` also works.
Direct fresh Freeze inputs can be `make`, a literal, bare `nil`, or a Clone result:
`im.FreezeSlice(make([]int, 2))`, `im.FreezeSlice([]int{1})`,
`im.FreezeSlice[int](nil)`, `im.FreezeSlice(frozen.Clone())`.
Each needs its own assignment/declaration; matching map forms work too.

| Rejected example | Replacement |
| --- | --- |
| `alias := values`, `values[1:]`, `&values[0]` | Read/copy elements; use Clone for independent mutable storage. |
| `values := []int(nil)` or `scores := map[string]int(nil)` | `var values []int` or `var scores map[string]int` |
| `type Ints []int; values := Ints(nil)` | `type Ints []int; var values Ints` |
| `im.FreezeSlice([]int(nil))` or `im.FreezeMap(map[string]int(nil))` | `im.FreezeSlice[int](nil)` or `im.FreezeMap[string, int](nil)` |
| `update(values)`, `return values` | Edit locally, then freeze and pass/return the frozen value. |
| `im.FreezeSlice(parameter)` | Construct a local builder first. |
| `values = append(values, other...)` | Append individual elements. |
| Freeze inside an active range over the same builder | Finish the range, then freeze. |
| Any use of `values` after Freeze | Use the frozen value, or Clone it. |

Do not use the old slice or map after Freeze, including in another branch or
loop iteration. A Clone of scalar elements, such as `int`, follows ordinary Go
rules if you never freeze it.

## Reads and Clone

Zero frozen values are empty: `var values im.FrozenSlice[int]` or
`values := im.FrozenSlice[int]{}`; the same forms work for `im.FrozenMap`.

| Allowed method | Example |
| --- | --- |
| Length | `n := frozen.Len()` |
| Slice element | `value := frozen.At(i)` |
| Map entry | `value, found := frozen.Get(key)` |
| Iteration | `for key, value := range frozen.All() { ... }` |
| Independent mutable copy | `next := frozen.Clone()` |

Call these methods on a local variable, a struct field, or a result from a named
function. Pass `im.FrozenSlice[T]` or `im.FrozenMap[K, V]` to functions by value.
Parentheses around a call or range expression are allowed.

Saving methods (`read := frozen.At`) or calling through the type
(`im.FrozenSlice[int].At(frozen, 0)`) fails; use `frozen.At(0)`.
Iterators carrying protected values require direct `range frozen.All()`.
Other iterators, such as those over `int` or `[2]int`, follow ordinary Go rules.
See [performance](performance.md) for operation costs.
