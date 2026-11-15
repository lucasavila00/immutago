# Frozen strings, slices, and maps

FrozenString, FrozenSlice, and FrozenMap satisfy the [Frozen type rule](types.md).
They can be copied, stored, and returned.
Their local projections follow the [projection rules](projections.md).

| Data | Stored value | Local projection | Constructor | Read locally |
| --- | --- | --- | --- | --- |
| String | `im.FrozenString` | `im.StringProjection` | `im.FreezeString(text)` | `frozen.Projection()` |
| Slice | `im.FrozenSlice[E]` | `im.SliceProjection[E]` | `im.FreezeSlice(values)` | `frozen.Projection()` |
| Map | `im.FrozenMap[K,V]` | `im.MapProjection[K,V]` | `im.FreezeMap(values)` | `frozen.Projection()` |

Each Projection method reads the stored value without copying its backing storage.
Slice elements and map keys and values must be Frozen types. Map keys must also
be Go-comparable.
Use the forms below with the [common forms](common-forms.md).

## Frozen strings

`FreezeString` wraps a native Go string without copying its bytes or consuming
the input. The native string remains usable because its bytes are immutable.
A substring input can retain its original allocation. Clone makes an explicit
independent copy.

```go
text := "standard"
frozen := im.FreezeString(text)
projection := frozen.Projection()
println(text, projection.Len(), projection.Is("standard"))
```

| String forms — exhaustive | Example |
| --- | --- |
| Zero declaration or empty literal | `var text im.FrozenString`, `im.FrozenString{}` |
| Wrap a native string | `text := im.FreezeString(source)` |
| Read byte count or one byte | `n := text.Len()`, `value := text.At(i)` |
| Compare with FrozenString or native string | `text.Equal(other)`, `text.EqualString("standard")` |
| Expose the shared native string | `fmt.Sprintf("%s", text.Unsafe())` |
| Read bytes in a loop | `for i := 0; i < text.Len(); i++ { println(text.At(i)) }` |
| Copy into independent native string storage | `saved := text.Clone()` |
| Create a local StringProjection | `projection := text.Projection()` |

Equal and EqualString use native string equality without allocating. Clone is
not needed to compare text. The zero value is empty. At has Go's normal bounds
panic. Unsafe lets the caller keep the shared bytes; the checker does not track
the returned native string. There is no substring operation or implicit string
conversion. For a native string projection, use `im.ProjectString(source)`.

## Frozen slices and maps

The checker rejects aliases and use of the original collection after Freeze:

1. Build a normal Go slice or map.
2. Freeze it without copying entries. Use only the frozen value after this call.
3. To make a changed version, Clone, edit the copy, and Freeze again.

## Example: change a score

Clone the frozen map, change the copy, and freeze it. The original map stays unchanged:

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

## Build before freezing

A **builder** is a local Go slice or map checked for exclusive use. These rules
apply before Freeze and to Go containers whose keys or elements contain
[protected storage](glossary.md).
Keep the slice or map in one local variable until Freeze. Clone copies the
slice or map; pointer elements still point to the same read-only objects.
An exclusive [Local builder](local.md) uses these operations but cannot be frozen.

| Allowed forms — exhaustive | Example |
| --- | --- |
| Declare from make, literal, zero, or Clone | `values := make([]int, 2)`, `values := []int{1}`, `var values []int`, `values := frozen.Clone()` |
| Initialize with bare `nil` | `var values []int = nil`, `var scores map[string]int = nil` |
| Read or replace an element | `x := values[i]`, `values[i] = x`, `scores[key] = x` |
| Change a scalar element or a field inside a Copied slice element | `counts[i]++`, `values[i].Score.Value++`, `values[i].Flags[0] = true` |
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
| `values[0].Set(2)` with a pointer receiver | `value := values[0]; value.Set(2); values[0] = value` |
| `values := []int(nil)` or `scores := map[string]int(nil)` | `var values []int` or `var scores map[string]int` |
| `type Ints []int; values := Ints(nil)` | `type Ints []int; var values Ints` |
| `im.FreezeSlice([]int(nil))` or `im.FreezeMap(map[string]int(nil))` | `im.FreezeSlice[int](nil)` or `im.FreezeMap[string, int](nil)` |
| `update(values)`, `return values` | Edit locally, then freeze and pass/return the frozen value. |
| `im.FreezeSlice(parameter)` | Construct a local builder first. |
| `values = append(values, other...)` | Append individual elements. |
| Freeze inside an active range over the same builder | Finish the range, then freeze. |
| Any use of `values` after Freeze | Use the frozen value, or Clone it. |
| `const n = len([1][]int{values})`, even though Go does not evaluate the array | Use `n := len(values)` before Freeze, or use the known constant `1`. |

Do not use the old slice or map after Freeze, including in another branch or
loop iteration. A Clone of Copied elements, including numbers and Copied records, follows
ordinary Go rules if you never freeze it.

## Slice and map reads

Zero frozen values are empty: `var values im.FrozenSlice[int]` or
`values := im.FrozenSlice[int]{}`; the same forms work for `im.FrozenMap`.

| Slice and map methods — exhaustive | Example |
| --- | --- |
| Length | `n := frozen.Len()` |
| Slice element | `value := frozen.At(i)` |
| Map entry | `value, found := frozen.Get(key)` |
| Iteration | `for key, value := range frozen.All() { ... }` |
| Local projection | `projection := frozen.Projection()` |
| Independent mutable copy | `next := frozen.Clone()` |

- Call methods on a local variable, a struct field, or a named function's result.
- Pass frozen collections to functions by value.
- Parentheses around a call or range expression are allowed.
- For iterators with protected values, use direct `range frozen.All()`.
- Other iterators, such as those over `int` or `[2]int`, use ordinary Go rules.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `text[0]`, `text[1:]`, or `string(text)` with FrozenString | Use At for a byte, Unsafe for a shared native string, or Clone for independent storage. |
| Return FrozenString from an exported service method | Return `text.Projection()`. |
| `read := frozen.At` | `value := frozen.At(0)` |
| `im.FrozenSlice[int].At(frozen, 0)` | `frozen.At(0)` |
