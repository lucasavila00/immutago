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

type SharedPoint struct { im.Immut; X int }
type LocalRefPoint struct { im.ImmutNoRef; X int }
type SharedLink struct { im.Immut; P *SharedPoint }
```

| Marker | Pointer policy |
| --- | --- |
| `im.Immut` | **Shareable pointers:** copy, store, pass to direct typed functions, or return them. |
| `im.ImmutNoRef` | **Local pointers:** keep them in local variables or pass them to direct typed non-retaining functions. Value copies can be stored and returned. |

In Rust terms, `Immut` resembles `Rc<T>`/`Arc<T>` with Go's GC managing lifetime;
`ImmutNoRef` resembles a scoped `&'a T`.

The allowed stored types are complete below. Every field, key, and element
must recursively match a row.

| Allowed form | Example |
| --- | --- |
| Boolean, integer, float, complex, or string | `bool`, `int`, `uintptr`, `float64`, `complex128`, `string` |
| Defined scalar | `type Count int` |
| Unnamed array of allowed elements | `[2]int`, `[2]SharedPoint` |
| Struct with one embedded marker and allowed fields | `SharedPoint`, `LocalRefPoint` above |
| Embedded marker alias | `type Mark = im.Immut; type Value struct { Mark; X int }` |
| Nested marked value | `type Pair struct { im.Immut; Child SharedPoint }` |
| Pointer to a concrete Immut struct | `*SharedPoint` |
| Recursive marked struct | `type SharedNode struct { im.Immut; Next *SharedNode }` |
| Frozen collection of allowed types | `im.FrozenSlice[SharedPoint]`, `im.FrozenMap[string, *SharedPoint]` |
| Alias or defined marked struct | `type Position = SharedPoint`, `type Position SharedPoint` |

Rejected type examples (**not exhaustive**):

| Rejected form | Replacement |
| --- | --- |
| `type Bad struct { Mark im.Immut }` | `type Good struct { im.Immut }` |
| `type Bad struct { *im.Immut }` | Embed `im.Immut` by value. |
| `type Bad struct { im.Immut; im.ImmutNoRef }` | Embed one marker. |
| `type Bad struct { Child SharedPoint }` | `type Good struct { im.Immut; Child SharedPoint }` |
| `type Bad struct { Children []SharedPoint }` | `type Good struct { im.Immut; Children im.FrozenSlice[SharedPoint] }` |
| `type Bad struct { im.Immut; P *LocalRefPoint }` | Store `LocalRefPoint` by value, or use `*SharedPoint` for sharing. |
| `type Counts [2]int` as a stored type | `type Counts struct { im.Immut; Values [2]int }` |

Allowed generic forms:

| Allowed form | Example |
| --- | --- |
| Fixed im types and functions | `im.FrozenSlice[SharedPoint]`, `im.FreezeSlice([]SharedPoint{})` |
| User generics without protected storage | `type Plain[T any] struct { Value T }`, used as `Plain[int]` |

Rejected generic examples (**not exhaustive**):

| Rejected form | Replacement |
| --- | --- |
| `type Box[T any] struct { im.Immut; Value T }` | `type Box struct { im.Immut; Value SharedPoint }` |
| `Plain[SharedPoint]` or `identity(point)` with generic `identity` | Use a concrete marked struct or copy `point` directly. |

## Construction and pointers

Prepare ordinary locals, then construct the object. Copies are also immutable.
Struct/array copies copy their fields; frozen wrappers and shared pointers keep
referring to the same read-only storage.

```go
x := 1
if condition { x = 2 }
point := SharedPoint{X: x}
point = SharedPoint{X: 3} // Whole-variable replacement is allowed.
p := &SharedPoint{X: 1}
q := p
p = &SharedPoint{X: 2} // q still observes X == 1.
```

### Pointers: both markers

These forms apply to both `*SharedPoint` (`im.Immut`) and
`*LocalRefPoint` (`im.ImmutNoRef`).

| Allowed form | Example |
| --- | --- |
| Declare a zero or nil pointer | `var p *SharedPoint`, `var p *LocalRefPoint = nil` |
| Construct fresh storage | `p := &SharedPoint{X: 1}`, `p := new(SharedPoint)` |
| Copy a value into fresh storage | `p := new(point)`, `p := new(SharedPoint{X: 1})` |
| Copy, rebind, or discard a pointer | `q := p`, `p = q`, `p = nil`, `_ = p` |
| Read a field or copy the whole value | `x := p.X`, `value := *p` |
| Compare pointers or nil | `same := p == q`, `if p != nil { ... }` |
| Pass to a direct function with an exact concrete parameter type | `read(p)`; `func read(p *SharedPoint) int { return p.X }` |

### Shareable pointers: `im.Immut` only

`*SharedPoint` also permits these forms:

| Allowed form | Example |
| --- | --- |
| Store in a marked field | `link := SharedLink{P: p}` |
| Store in an exclusive native builder | `values := []*SharedPoint{p}` |
| Store in a package variable | `saved = p` |
| Read a stored pointer | `p := link.P`, `p := values[i]` |
| Return or receive a pointer from a named function | `return p`, `p := makeSharedPoint()` |

Parentheses and aliases preserve these forms. Targets are concrete marked
structs. Pointers originate in the listed fresh constructions.

Rejected pointer examples (**not exhaustive**):

| Rejected example | Replacement |
| --- | --- |
| `point.X = 2`, `p.X = 2`, `*p = SharedPoint{}` | `point = SharedPoint{X: 2}` or `p = &SharedPoint{X: 2}` |
| `p := &point` | `p := new(point)` |
| `&point.X` | Read or copy the scalar. |
| `new(im.FrozenSlice[int])` | Pass the frozen wrapper by value. |
| `&[2]SharedPoint{}` | Put the array in a marked struct. |
| `[]*SharedPoint{{X: 1}}` | Use explicit allocation: `[]*SharedPoint{&SharedPoint{X: 1}}`. |
| `(*SharedPoint)(nil)` | `var p *SharedPoint = nil` |
| `return p` with `p *LocalRefPoint` | `return *p` with a `LocalRefPoint` result. |
| `saved = p` with `p *LocalRefPoint` | Store the value copy `*p`. |

## Statements and calls

Use the operations above inside these statement forms:

| Allowed form | Example |
| --- | --- |
| Declare or assign one destination | `value := SharedPoint{X: 1}`, `value = SharedPoint{X: 2}` |
| Change a native builder slot | `values[0] = value`, `counts[0]++`, `counts[0] += 2` |
| Bind a map reader's two results to new locals | `value, found := frozen.Get(key)` |
| Freeze in its own assignment or declaration | `frozen := im.FreezeSlice(values)`, `_ = im.FreezeSlice(values)` |
| Call a direct named function | `read(p)`, `n := read(p)` |
| Return values or `Immut` pointers, including multiple results | `return point`, `return p, true` |
| Forward named results directly | `return makePair()`, `takePair(makePair())` |
| Conditional or value switch | `if ready { p = makeSharedPoint() }`, `switch p.X { ... }` |
| Condition, three-clause, or unconditional loop | `for p.X > 0 { ... }`, `for i := 0; i < p.X; i++ { ... }`, `for { ... }` |
| Range with fresh loop variables | `for i, value := range values { ... }`, `for i := range count { ... }` |
| Assign one existing loop target | `for i = range count { ... }` |
| Use checked operations in a select case body | `case <-ready: values[0] = value` |

The one-destination rule applies to assignments and declarations, not returns.
`Get` is the only two-destination exception; both destinations must be new local
names or blanks. Blocks, `break`, and `continue` follow ordinary Go rules.
Whole package variables can also be reassigned; their objects remain immutable.
Calls are direct, synchronous, and concretely typed; dependencies are checked.

Rejected statement and call examples (**not exhaustive**):

| Rejected example | Replacement |
| --- | --- |
| `point, found := makePair()` | Return a marked result struct and bind it to one variable. |
| `value, p = SharedPoint{}, &SharedPoint{}` | Use two statements. |
| `for a[point.X], n = range values { ... }` | Bind fresh loop locals, then assign each destination separately. |
| `use(im.FreezeSlice(values))` | Bind Freeze to a local, then call `use`. |
| `case _, values[0] = <-ready:` | Receive first; write in the case body. |
| `case <-watch(im.FreezeSlice(values)):` | Freeze and compute the channel before the select; receive from the channel variable. |
| `value.Read()` or `reader.Read(p)` with protected receiver/parameters/results | Use a named function: `Read(p)`. |
| `var value any = p` | Use a concrete `*SharedPoint` variable. |
| `callback(p)` through a function variable with protected arguments | Use a direct named function. |
| `callback := func() int { return point.X }` | `x := point.X; callback := func() int { return x }` |
| `go read(p)` or `defer read(p)` | Call `read(p)` synchronously. |

## Frozen collections

Freeze consumes an exclusive native builder without copying. The old handle
cannot be used again. Frozen copies share storage; Clone creates an independent
native container for edits, sharing only immutable element pointers.

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
