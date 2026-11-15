# Common source forms

These forms apply to:

- Structs that embed `im.Shared` or `im.ByValue`.
- Shared frozen slices, maps, and strings.
- Local frozen views and marked values that contain them.

Combine them with each type's forms. [Owner](owner.md) has its own forms.
The [glossary](glossary.md) defines marked structs, protected storage, and builders.

## Markers and stored types

Embed exactly one marker by value:

```go
import "github.com/lucasavila00/immutago/pkg/im"

type SharedPoint struct { im.Shared; X int }
type ByValuePoint struct { im.ByValue; X int }
type SharedLink struct { im.Shared; P *SharedPoint }
```

Use the types below for struct fields and shared frozen collection keys and
elements. They also apply to each nested field, key, and element. Local frozen
collections use the smaller [view type list](local-views.md#type-arguments--exhaustive).
Unlisted protected types and operations are rejected.

| Allowed forms — exhaustive | Example |
| --- | --- |
| Scalar: boolean, integer, floating-point, or complex | `bool`, `int`, `uintptr`, `float64`, `complex128` |
| Defined type with one of those scalar kinds | `type Count int` |
| Shared frozen string | `im.FrozenString` |
| Empty marker value | `im.Shared`, `im.ByValue` |
| Unnamed array of allowed elements | `[2]int`, `[2]SharedPoint` |
| Struct with one embedded marker and allowed fields | `SharedPoint`, `ByValuePoint` above |
| Embedded marker alias | `type Mark = im.Shared; type Value struct { Mark; X int }` |
| Nested marked value, with either marker when the child contains no local view | `type LocalParent struct { im.ByValue; Child ByValuePoint }`, `type SharedParent struct { im.Shared; Child ByValuePoint }` |
| Pointer to a concrete Shared struct | `*SharedPoint` |
| Recursive marked struct | `type SharedNode struct { im.Shared; Next *SharedNode }` |
| Shared frozen collection of allowed types, without local views | `im.FrozenSlice[SharedPoint]`, `im.FrozenMap[im.FrozenString, *SharedPoint]` |
| Local frozen view or containing value, in a ByValue field or local unnamed array only | `type Result struct { im.ByValue; Text im.FrozenByValueString }` |
| Alias or defined marked struct | `type Position = SharedPoint`, `type Position SharedPoint` |
| Unchecked immutable value | `im.UncheckedValue[baked]`; see [Unchecked immutable values](unchecked-values.md). |
| Service reference | `supervise.Ref[Worker]`; see [service references](service-references.md). |

A ByValue value can contain shared data for persistent state, or local views
for a local result. When any field contains a view, the entire value has the
[local transport rules](local-views.md#transport--exhaustive). This includes
nested fields and array elements. Neither a Shared object nor a shared frozen
collection can contain such a value.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `type Bad struct { Mark im.Shared }` | `type Good struct { im.Shared }` |
| `type Bad struct { *im.Shared }` | Embed `im.Shared` by value. |
| `type Bad struct { im.Shared; im.ByValue }` | Embed one marker. |
| `type Bad struct { Child SharedPoint }` | `type Good struct { im.Shared; Child SharedPoint }` |
| `type Bad struct { Children []SharedPoint }` | `type Good struct { im.Shared; Children im.FrozenSlice[SharedPoint] }` |
| `type Bad struct { im.Shared; P *ByValuePoint }` | Store `ByValuePoint` by value, or use `*SharedPoint` for sharing. |
| `type Bad struct { im.ByValue; P *ByValuePoint }` | `type Good struct { im.ByValue; P ByValuePoint }` |
| `type Bad struct { im.Shared; Text string }` | Use `Text im.FrozenString`. |
| `type Bad struct { im.Shared; Text im.FrozenByValueString }` | Use `im.ByValue` for a local record, or store a shared string. |
| `type Counts [2]int` as a stored type | `type Counts struct { im.Shared; Values [2]int }` |

### Generics

| Allowed forms — exhaustive | Example |
| --- | --- |
| Fixed im types and functions | `im.FrozenSlice[SharedPoint]`, `im.FreezeSlice([]SharedPoint{})` |
| Fixed supervise APIs with protected values | `supervise.Service[ByValuePoint]`, `supervise.Start[Worker](root, args)`; saved arguments must use the stored types above and contain no local views. |
| User generics without protected storage | `type Plain[T any] struct { Value T }`, used as `Plain[int]` |

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `type Box[T any] struct { im.Shared; Value T }` | `type Box struct { im.Shared; Value SharedPoint }` |
| `Plain[SharedPoint]` or `identity(point)` with generic `identity` | Use a concrete marked struct or copy `point` directly. |

## Construction and scalar reads

```go
x := 1
if condition { x = 2 }
point := SharedPoint{X: x}
point = SharedPoint{X: 3} // Replace point with a new value.
p := &SharedPoint{X: 1}
q := p
p = &SharedPoint{X: 2} // q still observes X == 1.
```

| Allowed forms — exhaustive | Example |
| --- | --- |
| Zero value or literal | `var point SharedPoint`, `SharedPoint{X: 1}`, `[2]SharedPoint{}` |
| Copy a value, field, or array element | `other := point`, `x := point.X`, `child := points[i]` |
| Compare comparable values or arrays with `==` or `!=` | `same := point == other`, `same := points == otherPoints` |
| Array length and capacity | `len(points)`, `cap(points)` |
| Scalar unary `+`, `-`, `!`, `^`; scalar binary operators and conversions | `-point.X`, `point.X + 1`, `int64(point.X)` |
| Use scalar reads in literals or indices | `[]int{point.X}`, `a[point.X]` |
| Pass scalar reads to ordinary calls, including builtins, function variables, interface methods, and generics | `callback(point.X)`, `identity(point.X)` |

- After `x := point.X`, `x` is an ordinary integer; you can change it or capture it.
- The read of `point.X` still needs a listed statement form.
- Copies of marked values, pointers, frozen strings, and frozen collections keep their rules.
  Reading a local view from a field or element keeps the local restriction.
- Constant expressions follow the same rules. Use `count := len(points)` before
  a closure, then capture `count`; do not read `points` inside the closure.

## Common pointer forms

These forms apply to both `*SharedPoint` (`im.Shared`) and
`*ByValuePoint` (`im.ByValue`).

| Allowed forms — exhaustive | Example |
| --- | --- |
| Declare a zero or nil pointer | `var p *SharedPoint`, `var p *ByValuePoint = nil` |
| Create a new object | `p := &SharedPoint{X: 1}`, `p := new(SharedPoint)` |
| Create a new object from a value | `p := new(point)`, `p := new(SharedPoint{X: 1})` |
| Copy, replace, or discard a pointer | `q := p`, `p = q`, `p = nil`, `_ = p` |
| Read a field or copy the whole value | `x := p.X`, `value := *p` |
| Compare pointers or nil | `same := p == q`, `if p != nil { ... }` |
| Pass to a direct function with an exact concrete parameter type | `read(p)`; `func read(p *SharedPoint) int { return p.X }` |
| Call a method with the exact receiver type | `p.Read()`; `func (p *SharedPoint) Read() int { return p.X }` |

- Type aliases and parentheses preserve these forms.
- Pointer targets must be structs with `im.Shared` or `im.ByValue`.
- Construct pointers with the listed `new` or `&T{...}` forms.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `point.X = 2`, `p.X = 2`, `*p = SharedPoint{}` | `point = SharedPoint{X: 2}` or `p = &SharedPoint{X: 2}` |
| `p := &point` | `p := new(point)` |
| `&point.X` | Read or copy the scalar. |
| `new(im.FrozenSlice[int])` | Pass the frozen wrapper by value. |
| `&[2]SharedPoint{}` | Put the array in a marked struct. |
| `[]*SharedPoint{{X: 1}}` | Use explicit allocation: `[]*SharedPoint{&SharedPoint{X: 1}}`. |
| `(*SharedPoint)(nil)` | `var p *SharedPoint = nil` |

## Methods

Methods on protected receivers use these forms:

```go
func (p SharedPoint) XValue() int { return p.X }
func (p *SharedPoint) ReadX() int { return p.X }

type PointGroup struct { im.Shared; SharedPoint }

point := SharedPoint{X: 3}
p := &SharedPoint{X: 3}
group := PointGroup{SharedPoint: point}
point.XValue()
p.ReadX()
(*p).XValue()
group.SharedPoint.XValue()
```

For a protected receiver, the expression must have exactly the type in the
method declaration.
Combine these forms with the construction, read, argument, result, and statement
forms on this page.

| Allowed forms — exhaustive | Example |
| --- | --- |
| Value receiver on a marked struct | `func (p SharedPoint) XValue() int { return p.X }` |
| Pointer receiver on a marked struct | `func (p *ByValuePoint) ReadX() int { return p.X }` |
| Direct call on a value or pointer of the declared receiver type | `point.XValue()`, `p.ReadX()`, `(*p).XValue()` |
| Explicit field or element as receiver | `group.SharedPoint.XValue()`, `points[i].XValue()` |
| Constructed value, new pointer, or direct call result as receiver | `SharedPoint{X: 3}.XValue()`, `(&SharedPoint{X: 3}).ReadX()`, `makePoint().XValue()` |
| Direct method on an ordinary concrete receiver with protected inputs or results | `reader.Read(p)` with `func (r *Reader) Read(p *ByValuePoint) int`; `r` can have an ordinary mutable type. |
| Alias or parentheses in these forms | `type Position = SharedPoint`, `(p).ReadX()` |

| Method receiver | Body rules |
| --- | --- |
| Struct with `im.Shared` | Same checks as named functions; can return its pointer. |
| Struct with `im.ByValue` | Same checks as named functions; can return a value copy, but cannot store or return its pointer. |
| Either marked struct | Cannot change protected fields or have generic receivers. |
| Ordinary concrete receiver with protected inputs or results | Direct, non-generic calls. The protected inputs and results keep their im rules. |
| Fixed library type | Listed API forms. |

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `point.ReadX()` adds an implicit address | Construct `p := &SharedPoint{X: 3}`, then call `p.ReadX()`. |
| `p.XValue()` adds an implicit dereference | `(*p).XValue()` |
| `group.XValue()` selects a method from an embedded SharedPoint | `group.SharedPoint.XValue()` |
| `p.Counter.Inc()` calls a method on a protected scalar field | Copy the scalar first: `count := p.Counter; count.Inc()`. |
| `read := p.ReadX` or `(*SharedPoint).ReadX(p)` | `p.ReadX()` |
| `go p.ReadX()` or `defer p.ReadX()` | `p.ReadX()` |

## Statements and calls

Use the operations above inside these statement forms:

| Allowed forms — exhaustive | Example |
| --- | --- |
| Declare one variable or assign one destination | `var value SharedPoint`, `value := SharedPoint{X: 1}`, `value = SharedPoint{X: 2}` |
| Change a native builder slot | `values[0] = value`, `counts[0]++`, `counts[0] += 2` |
| Bind Start results to two local names or blanks with `:=`, `=`, or `var` | `ref, err := supervise.Start[Worker](root, args)`, `var ref, err = supervise.Start[Worker](root, args)` |
| Bind a map reader's two results to new locals | `value, found := frozen.Get(key)` |
| Freeze a slice or map in its own assignment or declaration | `frozen := im.FreezeSlice(values)`, `_ = im.FreezeSlice(values)` |
| Call a direct named function | `read(p)`, `n := read(p)` |
| Call a method in the list above | `n := p.ReadX()` |
| Return values or `Shared` pointers, including multiple results | `return point`, `return p, true` |
| Forward named results directly | `return makePair()`, `takePair(makePair())` |
| Conditional or value switch | `if ready { p = makeSharedPoint() }`, `switch p.X { ... }` |
| Condition, three-clause, or unconditional loop | `for p.X > 0 { ... }`, `for i := 0; i < p.X; i++ { ... }`, `for { ... }` |
| Range with fresh loop variables | `for i, value := range values { ... }`, `for i := range count { ... }` |
| Assign one existing writable loop target, using the assignment rules above | `for i = range count { ... }`, `for a[point.X] = range a { ... }` |
| Use checked operations in a select case body | `case <-ready: values[0] = value` |

- Assignments and declarations have one destination. Local views and values
  containing them cannot be package variables.
  Returns can have several results.
- `FrozenMap.Get` and `FrozenByValueMap.Get` permit two new local names or blanks.
- Calls to `supervise.Start` with protected arguments use these statement forms.
  Calls with only ordinary arguments use ordinary Go binding rules.
- Blocks, `break`, and `continue` follow ordinary Go rules.
- Named functions need the exact parameter type, such as `func read(p *SharedPoint) int`.
  Call directly, without `go` or `defer`.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `point, found := makePair()` | Return a marked result struct and bind it to one variable. |
| `value, p = SharedPoint{}, &SharedPoint{}` | Use two statements. |
| `Other(point)` where `type Other SharedPoint` | Construct `Other{X: point.X}`. |
| `OtherFrozen(frozen)` where `type OtherFrozen im.FrozenSlice[int]` | Keep the frozen type; use `Clone` for mutable storage. |
| `for a[point.X], n = range values { ... }` | Bind fresh loop locals, then assign each destination separately. |
| `use(im.FreezeSlice(values))` | Bind Freeze to a local, then call `use`. |
| `case _, values[0] = <-ready:` | Receive first; write in the case body. |
| `case <-watch(im.FreezeSlice(values)):` | Freeze and compute the channel before the select; receive from the channel variable. |
| `var value any = p` | Use a concrete `*SharedPoint` variable. |
| `callback(p)` through a function variable with protected arguments | Use a direct named function. |
| `callback := func() int { return point.X }` | `x := point.X; callback := func() int { return x }` |
| `func() int { return len([1]*ByValuePoint{p}) }` | Use `1`, or compute an ordinary length before the closure. |
| `const n = len([1]SharedPoint{})`, `[len([1]SharedPoint{})]int` | Use `const n = 1`, `[1]int`, or `n := len(points)` outside type syntax. |
| `go read(p)` or `defer read(p)` | Call `read(p)` synchronously. |
