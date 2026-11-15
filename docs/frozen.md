# Frozen collections

Use the shared frozen family when callers can keep the storage. Use the
local frozen family when callers need a checked read view.

| Data | Shared value | Local read view |
| --- | --- | --- |
| Slice | `im.FrozenSlice[T]` | `im.FrozenByValueSlice[T]` |
| Map | `im.FrozenMap[K, V]` | `im.FrozenByValueMap[K, V]` |
| String | `im.FrozenString` | `im.FrozenByValueString` |

A shared value can be copied, stored, and returned. Its copies share the whole
storage. A local view also shares storage; it is not an element-by-element copy.
Its type prevents persistent storage through the
[local view rules](local-views.md). The [string guide](frozen-strings.md)
lists string operations.

## Family support — exhaustive

These entries apply to all three types in each family. The linked guides give
their complete construction, read, and type-argument forms.

| Operation or placement | Shared `Frozen*` | Local `FrozenByValue*` |
| --- | --- | --- |
| Local copy, direct concrete argument, or return | Allowed. | Allowed; the result stays local. |
| Package variable or Shared field | Allowed. | Rejected, including inside another value. |
| ByValue field | Allowed. | Allowed; the containing record stays local. |
| Owner state, service start arguments, or shared frozen entries | Allowed. | Rejected, including inside another value. |
| Exported service method result | Rejected. | Allowed with the local type rules. |
| `.ByValue()` | Produces the matching local type with the same type arguments. | Not needed; already a view. |
| Clone | Copies native storage; shared references in entries stay shared. | Copies native storage when no entry contains a local view; copies text key bytes. |
| Native storage view, writable entry reference, or subview | Rejected. | Rejected. |

For either slice or map family, the checker rejects builder aliases and use of
the original collection after Freeze:

1. Build a normal Go slice or map.
2. Freeze it without copying entries. Use only the frozen value after this call.
3. To make a changed version, Clone, edit the copy, and Freeze again.

Shared keys and elements use the [stored types](common-forms.md#markers-and-stored-types).
Local views have a smaller [type list](local-views.md#type-arguments--exhaustive).
Use the forms below with the [common forms](common-forms.md). The lists are
complete; unlisted protected operations are rejected.

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

A **builder** is a local Go slice or map checked for exclusive use. These rules
apply before Freeze and to Go containers whose keys or elements contain
[protected storage](glossary.md).
Keep the slice or map in one local variable until Freeze. All forms in this
section apply to `FreezeSlice`, `FreezeMap`, `FreezeByValueSlice`, and
`FreezeByValueMap`, with the corresponding slice or map type.

| Allowed forms — exhaustive | Example |
| --- | --- |
| Declare from make, literal, zero, or Clone | `values := make([]int, 2)`, `values := []int{1}`, `var values []int`, `values := frozen.Clone()` |
| Initialize with bare `nil` | `var values []int = nil`, `var scores map[int]int = nil` |
| Read or replace an element | `x := values[i]`, `values[i] = x`, `scores[key] = x` |
| Read length or slice capacity | `len(values)`, `cap(values)` |
| Append individual elements to the same variable | `values = append(values, 1, 2)` |
| Delete, clear, or copy | `delete(scores, key)`, `clear(values)`, `copy(destination, source)` |
| Range over elements | `for i, value := range values { ... }` |
| Freeze the slice or map | `frozen := im.FreezeSlice(values)`, `view := im.FreezeByValueMap(scores)` |

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

These forms describe shared slices and maps. See the
[local view matrix](local-views.md#supported-operations--exhaustive) for local types.
Zero frozen values are empty: `var values im.FrozenSlice[int]` or
`values := im.FrozenSlice[int]{}`; the same forms work for `im.FrozenMap`.

| Allowed methods — exhaustive | Example |
| --- | --- |
| Length | `n := frozen.Len()` |
| Slice element | `value := frozen.At(i)` |
| Map entry | `value, found := frozen.Get(key)` |
| Iteration | `for key, value := range frozen.All() { ... }` |
| Independent mutable copy | `next := frozen.Clone()` |
| Local read view with the same type arguments | `view := frozen.ByValue()` |

- Call methods on a local variable, a struct field, or a named function's result.
- Pass frozen collections to functions by value.
- Parentheses around a call or range expression are allowed.
- For iterators with protected values, use direct `range frozen.All()`.
- Other iterators, such as those over `int` or `[2]int`, use ordinary Go rules.
- `ByValue` requires the local family's type arguments. For example,
  `FrozenSlice[int].ByValue()` is allowed; `FrozenSlice[FrozenString].ByValue()`
  is rejected. Use the [text-view builder](local-views.md#example-a-list-of-text-views).
- Shared Clone copies container slots. Pointers and nested shared frozen values
  still refer to the same immutable data. It does not make a recursive copy.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `read := frozen.At` | `value := frozen.At(0)` |
| `im.FrozenSlice[int].At(frozen, 0)` | `frozen.At(0)` |

See [performance](performance.md) for operation costs.
