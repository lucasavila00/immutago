# Frozen collections

`im.FrozenSlice[T]` wraps a Go slice; `im.FrozenMap[K, V]` wraps a Go map.
The checker rejects aliases and use of the original collection after Freeze.

1. Build a normal Go slice or map.
2. Freeze it without copying entries. Use only the frozen value after this call.
3. To make a changed version, Clone, edit the copy, and Freeze again.

Keys and elements must use the [allowed types](common-forms.md#markers-and-stored-types).
Use the forms below with the [common forms](common-forms.md).

## Example: change a score

Clone the frozen map, change the copy, and freeze it. The original map stays unchanged:

```go
package main

import "github.com/lucasavila00/immutago/pkg/im"

func main() {
    alice := im.FreezeString("alice")
    scores := map[im.FrozenString]int{alice: 10}
    frozen := im.FreezeMap(scores)

    next := frozen.Clone()
    next[alice] = 20
    updated := im.FreezeMap(next)

    oldScore, _ := frozen.Get(alice)
    newScore, _ := updated.Get(alice)
    println(oldScore, newScore) // 10 20
}
```

## Build before freezing

A **builder** is the Go slice or map that you prepare before Freeze. These rules
also apply to Go containers whose keys or elements contain protected storage.
Keep the slice or map in one local variable until Freeze. Clone copies the
slice or map; pointer elements still point to the same read-only objects.

| Allowed forms — exhaustive | Example |
| --- | --- |
| Declare from make, literal, zero, or Clone | `values := make([]int, 2)`, `values := []int{1}`, `var values []int`, `values := frozen.Clone()` |
| Initialize with bare `nil` | `var values []int = nil`, `var scores map[int]int = nil` |
| Read or replace an element | `x := values[i]`, `values[i] = x`, `scores[key] = x` |
| Read length or slice capacity | `len(values)`, `cap(values)` |
| Append individual elements to the same variable | `values = append(values, 1, 2)` |
| Delete, clear, or copy | `delete(scores, key)`, `clear(values)`, `copy(destination, source)` |
| Range over elements | `for i, value := range values { ... }` |
| Freeze the slice or map | `frozen := im.FreezeSlice(values)`, `frozen := im.FreezeMap(scores)` |

- `var` declarations also work.
- Both `copy` operands must satisfy the builder rules.
- Bare `nil` means Go's predeclared `nil`; `(nil)` also works.
- Freeze needs its own assignment or declaration.

Fresh inputs can go directly into Freeze; the same forms work for maps:

| Fresh Freeze inputs — exhaustive | Example |
| --- | --- |
| make | `frozen := im.FreezeSlice(make([]int, 2))` |
| Literal | `frozen := im.FreezeSlice([]int{1})` |
| Bare nil | `frozen := im.FreezeSlice[int](nil)` |
| Clone | `next := im.FreezeSlice(frozen.Clone())` |

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `alias := values`, `values[1:]`, `&values[0]` | Read/copy elements; use Clone for independent mutable storage. |
| `values := []int(nil)` or `scores := map[int]int(nil)` | `var values []int` or `var scores map[int]int` |
| `type Ints []int; values := Ints(nil)` | `type Ints []int; var values Ints` |
| `im.FreezeSlice([]int(nil))` or `im.FreezeMap(map[int]int(nil))` | `im.FreezeSlice[int](nil)` or `im.FreezeMap[int, int](nil)` |
| `update(values)`, `return values` | Edit locally, then freeze and pass/return the frozen value. |
| `im.FreezeSlice(parameter)` | Construct a local builder first. |
| `values = append(values, other...)` | Append individual elements. |
| Freeze inside an active range over the same builder | Finish the range, then freeze. |
| Any use of `values` after Freeze | Use the frozen value, or Clone it. |
| `const n = len([1][]int{values})`, even though Go does not evaluate the array | Use `n := len(values)` before Freeze, or use the known constant `1`. |

Do not use the old slice or map after Freeze, including in another branch or
loop iteration. A Clone of scalar elements, such as `int`, follows ordinary Go
rules if you never freeze it.

## Reads and Clone

Zero frozen values are empty: `var values im.FrozenSlice[int]` or
`values := im.FrozenSlice[int]{}`; the same forms work for `im.FrozenMap`.

| Allowed methods — exhaustive | Example |
| --- | --- |
| Length | `n := frozen.Len()` |
| Slice element | `value := frozen.At(i)` |
| Map entry | `value, found := frozen.Get(key)` |
| Iteration | `for key, value := range frozen.All() { ... }` |
| Independent mutable copy | `next := frozen.Clone()` |

- Call methods on a local variable, a struct field, or a named function's result.
- Pass frozen collections to functions by value.
- Parentheses around a call or range expression are allowed.
- For iterators with protected values, use direct `range frozen.All()`.
- Other iterators, such as those over `int` or `[2]int`, use ordinary Go rules.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `read := frozen.At` | `value := frozen.At(0)` |
| `im.FrozenSlice[int].At(frozen, 0)` | `frozen.At(0)` |

See [performance](performance.md) for operation costs.
