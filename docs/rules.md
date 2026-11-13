# Language rules

[Docs](README.md) · [Example](getting-started.md) · [API](../pkg/im/readme.md)

Objects are immutable from construction. Variables can be reassigned; their
objects cannot change. The allowed forms below are complete. Other protected
operations are rejected. Rejected examples are not exhaustive.

**Protected storage** means marked values, their fields, frozen collections,
and arrays containing them. Ordinary Go code outside these rules is unchanged.

## Markers and types

Embed exactly one marker by value:

```go
import "github.com/lucasavila00/immutago/pkg/im"

type Point struct { im.Immut; X int }
type Local struct { im.ImmutNoRef; X int }
type Link struct { im.Immut; P *Point }
```

| Marker | Reference policy |
| --- | --- |
| `im.Immut` | References can be copied, stored, passed to direct typed functions, and returned. |
| `im.ImmutNoRef` | References stay local or pass to direct typed non-retaining functions. No storage, returns, or captures. Value copies can be stored and returned. |

In Rust terms, `Immut` resembles `Rc<T>`/`Arc<T>` with Go's GC managing lifetime;
`ImmutNoRef` resembles a scoped `&'a T`. Neither uses NLL or mutable borrows.

Every stored field, key, and element must recursively match this table:

| Allowed type | Example |
| --- | --- |
| Boolean, integer, float, complex, or string | `bool`, `int`, `uintptr`, `float64`, `complex128`, `string` |
| Defined scalar | `type Count int` |
| Unnamed array of allowed elements | `[2]int`, `[2]Point` |
| Struct with exactly one embedded marker and allowed fields | `Point`, `Local` |
| Pointer to a concrete `Immut` struct with allowed fields | `*Point` |
| Frozen collection of allowed types | `im.FrozenSlice[Point]`, `im.FrozenMap[string, *Point]` |
| Alias or defined marked struct | `type Position = Point`, `type Position Point` |

Marker aliases work. Named marker fields, pointer-embedded markers, and structs
with both markers fail. Every nested struct needs its own marker. Unmarked
structs cannot contain protected storage, including through native containers.

Stored `*Local`, scalar pointers, native slices/maps, channels, interfaces, and
functions fail. Named arrays need a marked wrapper:

```go
// Rejected as a stored type: type Counts [2]int
// Allowed:
type Counts struct { im.Immut; Values [2]int }
type Node struct { im.Immut; Next *Node } // Recursive immutable types work.
```

Only the fixed `im` APIs handle protected storage generically. User generic
types, aliases, functions, and instantiations involving protected storage fail:

```go
// Rejected: type Box[T any] struct { im.Immut; Value T }
// Allowed:
type Box struct { im.Immut; Value Point }
```

Ordinary generics such as `Plain[int]` remain supported; `Plain[Point]` does not.

## Construction and references

Prepare ordinary locals, then construct the object. Copies are also immutable.
Struct/array copies copy their fields; frozen wrappers and shared pointers keep
referring to the same read-only storage.

```go
x := 1
if condition { x = 2 }
point := Point{X: x}
point = Point{X: 3} // Whole-variable replacement is allowed.
p := &Point{X: 1}
q := p
p = &Point{X: 2} // q still observes X == 1.
```

### What you can do with a reference

These common forms apply to both markers. For `Local`, every binding must stay
local; a called function must not retain its reference.

| Allowed form | Example |
| --- | --- |
| Declare a zero or nil pointer | `var p *Point`, `var p *Local = nil` |
| Construct fresh storage | `p := &Point{X: 1}`, `p := new(Point)` |
| Copy a value into fresh storage | `p := new(point)`, `p := new(Point{X: 1})` |
| Copy, rebind, or discard a pointer | `q := p`, `p = q`, `p = nil`, `_ = p` |
| Read a field or copy the whole value | `x := p.X`, `value := *p` |
| Compare pointers or nil | `same := p == q`, `if p != nil { ... }` |
| Pass to a direct function with an exact concrete parameter type | `read(p)`; `func read(p *Point) int { return p.X }` |

`Immut` additionally permits these forms:

| Allowed form | Example |
| --- | --- |
| Store in a marked field | `link := Link{P: p}` |
| Store in an exclusive native builder | `values := []*Point{p}` |
| Store in a package variable | `saved = p` |
| Read a stored reference | `p := link.P`, `p := values[i]` |
| Return or receive a reference from a named function | `return p`, `p := makePoint()` |

Parentheses and aliases preserve these forms. Pointer targets must be concrete
marked structs, never arrays, frozen wrappers, scalars, or other pointers.
References originate only in fresh construction, not `&point` or `&owner.Child`:
reassigning addressed storage would change what an existing reference observes.

| Rejected example | Replacement |
| --- | --- |
| `point.X = 2`, `p.X = 2`, `*p = Point{}` | `point = Point{X: 2}` or `p = &Point{X: 2}` |
| `p := &point` | `p := new(point)` |
| `&point.X` | Read or copy the scalar. |
| `new(im.FrozenSlice[int])` | Pass the frozen wrapper by value. |
| `&[2]Point{}` | Put the array in a marked struct. |
| `[]*Point{{X: 1}}` | Use explicit allocation: `[]*Point{&Point{X: 1}}`. |
| `(*Point)(nil)` | `var p *Point = nil` |
| Returning or storing `p` where `p` is `*Local` | Return or store the value copy `*p`. |

## Statements and calls

Use the operations above inside these statement forms:

| Allowed form | Example |
| --- | --- |
| Declare or assign one destination | `value := Point{X: 1}`, `value = Point{X: 2}` |
| Change a native builder slot | `values[0] = value`, `counts[0]++`, `counts[0] += 2` |
| Bind a map reader's two results to new locals | `value, found := frozen.Get(key)` |
| Freeze in its own assignment or declaration | `frozen := im.FreezeSlice(values)`, `_ = im.FreezeSlice(values)` |
| Call a direct named function | `read(p)`, `n := read(p)` |
| Return values or `Immut` references, including multiple results | `return point`, `return p, true` |
| Forward named results directly | `return makePair()`, `takePair(makePair())` |
| Use `if`, ordinary loops, or value switches | `if ready { p = makePoint() }`, `for i := range values { ... }`, `switch p.X { ... }` |
| Assign one existing loop target | `for i = range count { ... }` |
| Use checked operations in a select case body | `case <-ready: values[0] = value` |

The one-destination rule applies to assignments and declarations, not returns.
`Get` is the only two-destination exception; both destinations must be new local
names or blanks. Blocks, `break`, and `continue` follow ordinary Go rules.
Whole package variables can also be reassigned; their objects remain immutable.

| Rejected example | Replacement |
| --- | --- |
| `point, found := makePair()` | Return a marked result struct and bind it to one variable. |
| `value, p = Point{}, &Point{}` | Use two statements. |
| `for a[point.X], n = range values { ... }` | Bind fresh loop locals, then assign each destination separately. |
| `use(im.FreezeSlice(values))` | Bind Freeze to a local, then call `use`. |
| `case _, values[0] = <-ready:` | Receive first; write in the case body. |
| `case <-watch(im.FreezeSlice(values)):` | Freeze and compute the channel before the select; receive from the channel variable. |

Protected calls are direct, synchronous, and concrete. Methods on protected
receivers (including promoted methods), interface transport, protected callbacks, closure captures,
`go`, and `defer` are unsupported. Dependencies are checked too.

```go
// Rejected: callback := func() int { return point.X }
// Allowed: pass a scalar copy to ordinary callback code.
x := point.X
callback := func() int { return x }
```

## Frozen collections

Freeze consumes an exclusive native builder without copying. The old handle
cannot be used again. Frozen copies share storage; Clone creates an independent
native container for edits, sharing only immutable element references.

```go
scores := map[string]int{"alice": 10}
frozen := im.FreezeMap(scores)
next := frozen.Clone()
next["alice"] = 20
updated := im.FreezeMap(next) // frozen still contains 10.
```

### Builders

These rules apply to any native map/slice you freeze, and to native containers
whose keys or elements contain protected storage.

| Allowed form | Example |
| --- | --- |
| Declare from make, literal, zero, nil, or Clone | `values := make([]int, 2)`, `values := []int{1}`, `var values []int`, `values := frozen.Clone()` |
| Read or replace a slot | `x := values[i]`, `values[i] = x`, `scores[key] = x` |
| Read length or slice capacity | `len(values)`, `cap(values)` |
| Append individual elements to the same variable | `values = append(values, 1, 2)` |
| Delete, clear, or copy | `delete(scores, key)`, `clear(values)`, `copy(destination, source)` |
| Range over elements | `for i, value := range values { ... }` |
| Consume with Freeze | `frozen := im.FreezeSlice(values)`, `frozen := im.FreezeMap(scores)` |

`var` declarations also work. `copy` operands must satisfy the builder rules.
Direct fresh Freeze inputs can be `make`, a literal, nil, or a Clone result:
`im.FreezeSlice(make([]int, 2))`, `im.FreezeSlice([]int{1})`,
`im.FreezeSlice[int](nil)`, `im.FreezeSlice(frozen.Clone())`.
Each needs its own assignment/declaration; matching map forms work too.

| Rejected example | Replacement |
| --- | --- |
| `alias := values`, `values[1:]`, `&values[0]` | Read/copy elements; use Clone for independent mutable storage. |
| `update(values)`, `return values` | Edit locally, then freeze and pass/return the frozen value. |
| `im.FreezeSlice(parameter)` | Construct a local builder first. |
| `values = append(values, other...)` | Append individual elements. |
| Freeze inside an active range over the same builder | Finish the range, then freeze. |
| Any use of `values` after Freeze | Use the frozen value, or Clone it. |

A branch or loop cannot make a consumed handle usable again. Ordinary scalar
clones that are never frozen follow ordinary Go container rules.

### Reads and Clone

| Allowed method | Example |
| --- | --- |
| Length | `n := frozen.Len()` |
| Slice element | `value := frozen.At(i)` |
| Map entry | `value, found := frozen.Get(key)` |
| Iteration | `for key, value := range frozen.All() { ... }` |
| Independent mutable copy | `next := frozen.Clone()` |

Receivers can be frozen locals, fields, or named-call results. Pass wrappers
by value. Parentheses preserve direct calls and ranges.

Saving methods (`read := frozen.At`) or calling through the type
(`im.FrozenSlice[int].At(frozen, 0)`) fails; use `frozen.At(0)`.
Iterators carrying protected values require direct `range frozen.All()`.
Other iterators, such as those over `int` or `[2]int`, follow ordinary Go rules.
See [performance](performance.md) for operation costs.

## Check the program

Run `immutago ./...` or `go vet -vettool=/absolute/path/to/immutago ./...`.
`go build` alone does not enforce these rules. There are no suppression or trust
annotations. Unsupported forms fail rather than being skipped.

Checked code cannot import `unsafe` or `reflect`, or declare functions without
Go bodies—even for scalar-only functions. Replace `func readCount() int` with
a checked Go implementation such as `func readCount() int { return 1 }`.
