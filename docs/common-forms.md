# Common source forms

[Language guide](rules.md) · [Shared](shared.md) · [Scoped](scoped.md) · [Frozen collections](frozen.md)

These forms apply to marked objects and frozen collections. Combine them with
the forms on the relevant type's page. The lists are complete; other protected
operations are rejected. Rejected examples are not exhaustive.
[Owner slots and holders](owner.md) have their own forms.

[Types](#markers-and-stored-types) · [Values](#construction-and-scalar-reads) ·
[Pointers](#common-pointer-forms) · [Statements](#statements-and-calls)

## Markers and stored types

Embed exactly one marker by value:

```go
import "github.com/lucasavila00/immutago/pkg/im"

type SharedPoint struct { im.Shared; X int }
type ScopedPoint struct { im.Scoped; X int }
type SharedLink struct { im.Shared; P *SharedPoint }
```

The allowed stored types are complete below. Every field, key, and element
must recursively match a row.

| Allowed form | Example |
| --- | --- |
| Boolean, integer, float, complex, or string | `bool`, `int`, `uintptr`, `float64`, `complex128`, `string` |
| Defined scalar | `type Count int` |
| Unnamed array of allowed elements | `[2]int`, `[2]SharedPoint` |
| Struct with one embedded marker and allowed fields | `SharedPoint`, `ScopedPoint` above |
| Embedded marker alias | `type Mark = im.Shared; type Value struct { Mark; X int }` |
| Nested marked value, with either marker on the parent or child | `type LocalParent struct { im.Scoped; Child ScopedPoint }`, `type SharedParent struct { im.Shared; Child ScopedPoint }` |
| Pointer to a concrete Shared struct | `*SharedPoint` |
| Recursive marked struct | `type SharedNode struct { im.Shared; Next *SharedNode }` |
| Frozen collection of allowed types | `im.FrozenSlice[SharedPoint]`, `im.FrozenMap[string, *SharedPoint]` |
| Alias or defined marked struct | `type Position = SharedPoint`, `type Position SharedPoint` |

Rejected type examples (**not exhaustive**):

| Rejected form | Replacement |
| --- | --- |
| `type Bad struct { Mark im.Shared }` | `type Good struct { im.Shared }` |
| `type Bad struct { *im.Shared }` | Embed `im.Shared` by value. |
| `type Bad struct { im.Shared; im.Scoped }` | Embed one marker. |
| `type Bad struct { Child SharedPoint }` | `type Good struct { im.Shared; Child SharedPoint }` |
| `type Bad struct { Children []SharedPoint }` | `type Good struct { im.Shared; Children im.FrozenSlice[SharedPoint] }` |
| `type Bad struct { im.Shared; P *ScopedPoint }` | Store `ScopedPoint` by value, or use `*SharedPoint` for sharing. |
| `type Bad struct { im.Scoped; P *ScopedPoint }` | `type Good struct { im.Scoped; P ScopedPoint }` |
| `type Counts [2]int` as a stored type | `type Counts struct { im.Shared; Values [2]int }` |

Allowed generic forms:

| Allowed form | Example |
| --- | --- |
| Fixed im types and functions | `im.FrozenSlice[SharedPoint]`, `im.FreezeSlice([]SharedPoint{})` |
| User generics without protected storage | `type Plain[T any] struct { Value T }`, used as `Plain[int]` |

Rejected generic examples (**not exhaustive**):

| Rejected form | Replacement |
| --- | --- |
| `type Box[T any] struct { im.Shared; Value T }` | `type Box struct { im.Shared; Value SharedPoint }` |
| `Plain[SharedPoint]` or `identity(point)` with generic `identity` | Use a concrete marked struct or copy `point` directly. |

## Construction and scalar reads

Prepare ordinary locals, then construct the object. Copies of marked values and
frozen collections stay immutable.
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

| Allowed form | Example |
| --- | --- |
| Zero value or literal | `var point SharedPoint`, `SharedPoint{X: 1}`, `[2]SharedPoint{}` |
| Copy a value, field, or array element | `other := point`, `x := point.X`, `child := points[i]` |
| Compare comparable values or arrays with `==` or `!=` | `same := point == other`, `same := points == otherPoints` |
| Array length and capacity | `len(points)`, `cap(points)` |
| Scalar unary `+`, `-`, `!`, `^`; scalar binary operators and conversions | `-point.X`, `point.X + 1`, `int64(point.X)` |
| Use scalar reads in literals or indices; index or slice strings | `[]int{point.X}`, `a[point.X]`, `user.Name[1:]` |
| Pass scalar reads to ordinary calls, including builtins, function variables, interface methods, and generics | `callback(point.X)`, `identity(point.X)` |

A scalar read produces ordinary Go data. The statement that reads protected
storage must still match the statement list below. After `x := point.X`, uses
of `x` follow ordinary Go rules. Copies that contain marked values, pointers, or frozen collections remain protected.
Compiler-constant operands, such as the array in `len([1]SharedPoint{})`, do
not perform runtime reads or create storage.

Strings also follow ordinary Go rules, including when read from a marked field.
Ordinary library APIs are trusted to keep string bytes immutable. There is no
string provenance tracking or automatic copy. See the [package boundary](rules.md#package-boundary).

## Common pointer forms

These forms apply to both `*SharedPoint` (`im.Shared`) and
`*ScopedPoint` (`im.Scoped`).

| Allowed form | Example |
| --- | --- |
| Declare a zero or nil pointer | `var p *SharedPoint`, `var p *ScopedPoint = nil` |
| Construct fresh storage | `p := &SharedPoint{X: 1}`, `p := new(SharedPoint)` |
| Copy a value into fresh storage | `p := new(point)`, `p := new(SharedPoint{X: 1})` |
| Copy, rebind, or discard a pointer | `q := p`, `p = q`, `p = nil`, `_ = p` |
| Read a field or copy the whole value | `x := p.X`, `value := *p` |
| Compare pointers or nil | `same := p == q`, `if p != nil { ... }` |
| Pass to a direct function with an exact concrete parameter type | `read(p)`; `func read(p *SharedPoint) int { return p.X }` |

Parentheses and aliases preserve the pointer forms here and on the marker pages.
Targets are concrete marked structs. Pointers originate in the listed fresh
constructions.

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

## Statements and calls

Use the operations above inside these statement forms:

| Allowed form | Example |
| --- | --- |
| Declare or assign one destination | `value := SharedPoint{X: 1}`, `value = SharedPoint{X: 2}` |
| Change a native builder slot | `values[0] = value`, `counts[0]++`, `counts[0] += 2` |
| Bind a map reader's two results to new locals | `value, found := frozen.Get(key)` |
| Freeze in its own assignment or declaration | `frozen := im.FreezeSlice(values)`, `_ = im.FreezeSlice(values)` |
| Call a direct named function | `read(p)`, `n := read(p)` |
| Return values or `Shared` pointers, including multiple results | `return point`, `return p, true` |
| Forward named results directly | `return makePair()`, `takePair(makePair())` |
| Conditional or value switch | `if ready { p = makeSharedPoint() }`, `switch p.X { ... }` |
| Condition, three-clause, or unconditional loop | `for p.X > 0 { ... }`, `for i := 0; i < p.X; i++ { ... }`, `for { ... }` |
| Range with fresh loop variables | `for i, value := range values { ... }`, `for i := range count { ... }` |
| Assign one existing writable loop target, using the assignment rules above | `for i = range count { ... }`, `for a[point.X] = range a { ... }` |
| Use checked operations in a select case body | `case <-ready: values[0] = value` |

The one-destination rule applies to assignments and declarations, not returns.
`Get` is the only two-destination exception; both destinations must be new local
names or blanks. Blocks, `break`, and `continue` follow ordinary Go rules.
Whole package variables can also be reassigned; their objects remain immutable.
Calls carrying protected values or pointers must be direct, synchronous, and
concretely typed; packages using these contracts are checked under the
[package boundary](rules.md#package-boundary).

Rejected statement and call examples (**not exhaustive**):

| Rejected example | Replacement |
| --- | --- |
| `point, found := makePair()` | Return a marked result struct and bind it to one variable. |
| `value, p = SharedPoint{}, &SharedPoint{}` | Use two statements. |
| `Other(point)` where `type Other SharedPoint` | Construct `Other{X: point.X}`. |
| `OtherFrozen(frozen)` where `type OtherFrozen im.FrozenSlice[int]` | Keep the frozen type; use `Clone` for mutable storage. |
| `for a[point.X], n = range values { ... }` | Bind fresh loop locals, then assign each destination separately. |
| `use(im.FreezeSlice(values))` | Bind Freeze to a local, then call `use`. |
| `case _, values[0] = <-ready:` | Receive first; write in the case body. |
| `case <-watch(im.FreezeSlice(values)):` | Freeze and compute the channel before the select; receive from the channel variable. |
| `value.Read()` or `reader.Read(p)` with protected receiver/parameters/results | Use a named function: `Read(p)`. |
| `var value any = p` | Use a concrete `*SharedPoint` variable. |
| `callback(p)` through a function variable with protected arguments | Use a direct named function. |
| `callback := func() int { return point.X }` | `x := point.X; callback := func() int { return x }` |
| `go read(p)` or `defer read(p)` | Call `read(p)` synchronously. |
