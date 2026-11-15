# Common source forms

These forms apply to:

- Structs that embed `im.Frozen` or `im.Local`.
- Named interfaces that embed [im.FrozenInterface](frozen-interfaces.md).
- `im.FrozenString`, `im.FrozenSlice`, `im.FrozenMap`, and local read projections.

Combine them with each type's forms. [Cell](cell.md) has its own forms.
Ordinary [Copied values](copied.md) and their ordinary holders use all Go forms.
Copied data inside protected storage keeps the owner's rules on this page.
The [glossary](glossary.md) defines marked structs, protected storage, and builders.

## Markers and stored types

The [type rules](types.md) define Frozen, Copied, Local, and read results.
Every package that declares Frozen records uses the
[required generation step](rules.md#required-generation).
Embed exactly one marker by value:

```go
type FrozenPoint struct { im.Frozen; X int }
type LocalPoint struct { im.Local; X int }
type CopiedPoint struct { im.Copied; X int }
type FrozenLink struct { im.Frozen; P *FrozenPoint }
```

Frozen fields and shared frozen entries require Frozen types. Local fields
require read results. Copied fields use the complete [value-only field list](copied.md#fields--exhaustive).
A Local value cannot be hidden in persistent storage.

### Generics

| Allowed forms — exhaustive | Example |
| --- | --- |
| Fixed im types and functions | `im.FrozenSlice[FrozenPoint]`, `im.FreezeSlice([]FrozenPoint{})` |
| Fixed supervise APIs with protected values | `supervise.Service[FrozenPoint]`, `supervise.Start[Worker](root, args)`; saved protected arguments must be Frozen values, never Local values. |
| User generics without protected storage | `type Plain[T any] struct { Value T }`, used as `Plain[int]` |

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `type Box[T any] struct { im.Frozen; Value T }` | `type Box struct { im.Frozen; Value FrozenPoint }` |
| `Plain[FrozenPoint]` or `identity(point)` with generic `identity` | Use a concrete marked struct or copy `point` directly. |

## Construction and scalar reads

```go
x := 1
if condition { x = 2 }
point := FrozenPoint{X: x}
point = FrozenPoint{X: 3} // Replace point with a new value.
p := &FrozenPoint{X: 1}
q := p
p = &FrozenPoint{X: 2} // q still observes X == 1.
```

| Allowed forms — exhaustive | Example |
| --- | --- |
| Zero value or literal | `var point FrozenPoint`, `FrozenPoint{X: 1}`, `[2]FrozenPoint{}` |
| Copy a value, field, or array element | `other := point`, `x := point.X`, `child := points[i]` |
| Compare comparable values or arrays with `==` or `!=` | `same := point == other`, `same := points == otherPoints` |
| Array length and capacity | `len(points)`, `cap(points)` |
| Scalar unary `+`, `-`, `!`, `^`; scalar binary operators and conversions | `-point.X`, `point.X + 1`, `int64(point.X)` |
| Use scalar or Copied field reads in literals | `[]int{point.X}`, `[]Score{point.Score}` |
| Use scalar reads in indices; index or slice strings | `a[point.X]`, `user.Name[1:]` |
| Pass scalar or Copied field reads to ordinary calls, including builtins, function variables, interface methods, and generics | `callback(point.X)`, `callback(point.Score)`, `identity(point.Score)` |
| Store a scalar or Copied field read in an ordinary interface | `var value any = point.Score` |

- After `x := point.X`, `x` is an ordinary integer; you can change it or capture it.
- The read of `point.X` still needs a listed statement form.
- Frozen and Local copies, protected pointers, and frozen collections keep their rules.
- Constant expressions follow the same rules. Use `count := len(points)` before
  a closure, then capture `count`; do not read `points` inside the closure.

## Common pointer forms

These forms apply to both `*FrozenPoint` (`im.Frozen`) and
`*LocalPoint` (`im.Local`).

| Allowed forms — exhaustive | Example |
| --- | --- |
| Declare a zero or nil pointer | `var p *FrozenPoint`, `var p *LocalPoint = nil` |
| Create a new object | `p := &FrozenPoint{X: 1}`, `p := new(FrozenPoint)` |
| Create a new object from a value | `p := new(point)`, `p := new(FrozenPoint{X: 1})` |
| Copy, replace, or discard a pointer | `q := p`, `p = q`, `p = nil`, `_ = p` |
| Read a field or copy the whole value | `x := p.X`, `value := *p` |
| Compare pointers or nil | `same := p == q`, `if p != nil { ... }` |
| Pass to a direct function with an exact concrete parameter type | `read(p)`; `func read(p *FrozenPoint) int { return p.X }` |
| Call a method with the exact receiver type | `p.Read()`; `func (p *FrozenPoint) Read() int { return p.X }` |

- Type aliases and parentheses preserve these forms.
- Pointer targets must be structs with `im.Frozen` or `im.Local`.
- Construct pointers with the listed `new` or `&T{...}` forms.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `point.X = 2`, `p.X = 2`, `*p = FrozenPoint{}` | `point = FrozenPoint{X: 2}` or `p = &FrozenPoint{X: 2}` |
| `p := &point` | `p := new(point)` |
| `&point.X` | Read or copy the scalar. |
| `new(im.FrozenSlice[int])` | Pass the frozen wrapper by value. |
| `&[2]FrozenPoint{}` | Put the array in a marked struct. |
| `[]*FrozenPoint{{X: 1}}` | Use explicit allocation: `[]*FrozenPoint{&FrozenPoint{X: 1}}`. |
| `(*FrozenPoint)(nil)` | `var p *FrozenPoint = nil` |

## Methods

Methods on protected receivers use these forms:

```go
func (p FrozenPoint) XValue() int { return p.X }
func (p *FrozenPoint) ReadX() int { return p.X }

type PointGroup struct { im.Frozen; FrozenPoint }

point := FrozenPoint{X: 3}
p := &FrozenPoint{X: 3}
group := PointGroup{FrozenPoint: point}
point.XValue()
p.ReadX()
(*p).XValue()
group.FrozenPoint.XValue()
```

For a protected receiver, the expression must have exactly the type in the
method declaration.
Combine these forms with the construction, read, argument, result, and statement
forms on this page.

| Allowed forms — exhaustive | Example |
| --- | --- |
| Value receiver on a marked struct | `func (p FrozenPoint) XValue() int { return p.X }` |
| Pointer receiver on a marked struct | `func (p *LocalPoint) ReadX() int { return p.X }` |
| Direct call on a value or pointer of the declared receiver type | `point.XValue()`, `p.ReadX()`, `(*p).XValue()` |
| Explicit field or element as receiver | `group.FrozenPoint.XValue()`, `points[i].XValue()` |
| Direct fixed supervise method on a protected field or element | `route.Worker.IsZero()`, `route.Workers[i].IsZero()` for Ref fields and elements; the [service-reference rules](service-references.md) apply. |
| Constructed value, new pointer, or direct call result as receiver | `FrozenPoint{X: 3}.XValue()`, `(&FrozenPoint{X: 3}).ReadX()`, `makePoint().XValue()` |
| Direct method on an ordinary concrete receiver with protected inputs or results | `reader.Read(p)` with `func (r *Reader) Read(p *LocalPoint) int`; `r` can have an ordinary mutable type. |
| Direct call through a proved Frozen interface | `scorer.Score(features)`; use the [complete interface rules](frozen-interfaces.md). |
| Alias or parentheses in these forms | `type Position = FrozenPoint`, `(p).ReadX()` |

| Method receiver | Body rules |
| --- | --- |
| Struct with `im.Frozen` | Same checks as named functions; can return its pointer. |
| Struct with `im.Local` | Its value stays Local. Its pointer can only be used in local direct calls. |
| Frozen or Local record | Cannot change protected fields or have generic receivers. |
| Copied field inside a protected owner | Exact value-receiver calls copy the field. Pointer-receiver calls that address the field are rejected. |
| Ordinary concrete receiver with protected inputs or results | Direct, non-generic calls. The protected inputs and results keep their im rules. |
| Fixed library type | Listed API forms. |

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `point.ReadX()` adds an implicit address | Construct `p := &FrozenPoint{X: 3}`, then call `p.ReadX()`. |
| `p.XValue()` adds an implicit dereference | `(*p).XValue()` |
| `group.XValue()` selects a method from an embedded FrozenPoint | `group.FrozenPoint.XValue()` |
| `p.Counter.Inc()` calls a method on a protected scalar field | Copy the scalar first: `count := p.Counter; count.Inc()`. |
| `read := p.ReadX` or `(*FrozenPoint).ReadX(p)` | `p.ReadX()` |
| `go p.ReadX()` or `defer p.ReadX()` | `p.ReadX()` |

## Statements and calls

Use the operations above inside these statement forms:

| Allowed forms — exhaustive | Example |
| --- | --- |
| Declare one variable or assign one destination | `var value FrozenPoint`, `value := FrozenPoint{X: 1}`, `value = FrozenPoint{X: 2}` |
| Change a native builder slot or its Copied fields | `values[0] = value`, `counts[0]++`, `counts[0] += 2`, `scores[0].Value++` |
| Bind Start results to two local names or blanks with `:=`, `=`, or `var` | `ref, err := supervise.Start[Worker](root, args)`, `var ref, err = supervise.Start[Worker](root, args)` |
| Bind `(read result, error)` from a direct named function or concrete method to two local names or blanks | `value, err := ref.Borrow().Read()`, `value, err = read()`, `var value, err = read()` |
| Bind a map Get call's two results to new locals | `value, found := frozen.Get(key)` |
| Freeze a slice or map in its own assignment or declaration | `frozen := im.FreezeSlice(values)`, `_ = im.FreezeSlice(values)` |
| Call a direct named function | `read(p)`, `n := read(p)` |
| Call a method in the list above | `n := p.ReadX()` |
| Return values or Frozen pointers, including multiple results | `return point`, `return p, true` |
| Forward named results directly | `return makePair()`, `takePair(makePair())` |
| Conditional or value switch | `if ready { p = makeFrozenPoint() }`, `switch p.X { ... }` |
| Condition, three-clause, or unconditional loop | `for p.X > 0 { ... }`, `for i := 0; i < p.X; i++ { ... }`, `for { ... }` |
| Range with fresh loop variables | `for i, value := range values { ... }`, `for i := range count { ... }` |
| Assign one existing writable loop target, using the assignment rules above | `for i = range count { ... }`, `for a[point.X] = range a { ... }` |
| Use checked operations in a select case body | `case <-ready: values[0] = value` |

- Assignments and declarations have one destination, except for the result-pair
  forms above. Returns can have several results.
- FrozenMap.Get and MapProjection.Get permit two new local names or blanks.
- Calls to `supervise.Start` with protected arguments use these statement forms.
  Calls with only ordinary arguments use ordinary Go binding rules.
- Result/error bindings use the [service result signature](service-results.md),
  including direct typed helper calls. Calls with only ordinary values keep Go rules.
- Blocks, `break`, and `continue` follow ordinary Go rules.
- Named functions need the exact parameter type, such as `func read(p *FrozenPoint) int`.
  Call directly, without `go` or `defer`.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `point, found := makePair()` returning `(FrozenPoint, bool)` | Return a marked result struct and bind it to one variable. |
| `value, p = FrozenPoint{}, &FrozenPoint{}` | Use two statements. |
| `Other(point)` where `type Other FrozenPoint` | Construct `Other{X: point.X}`. |
| `OtherFrozen(frozen)` where `type OtherFrozen im.FrozenSlice[int]` | Keep the frozen type; use `Clone` for mutable storage. |
| `for a[point.X], n = range values { ... }` | Bind fresh loop locals, then assign each destination separately. |
| `use(im.FreezeSlice(values))` | Bind Freeze to a local, then call `use`. |
| `case _, values[0] = <-ready:` | Receive first; write in the case body. |
| `case <-watch(im.FreezeSlice(values)):` | Freeze and compute the channel before the select; receive from the channel variable. |
| `var value any = p` | Use a concrete `*FrozenPoint` variable. |
| `callback(p)` through a function variable with protected arguments | Use a direct named function. |
| `callback := func() int { return point.X }` | `x := point.X; callback := func() int { return x }` |
| `func() int { return len([1]*LocalPoint{p}) }` | Use `1`, or compute an ordinary length before the closure. |
| `const n = len([1]FrozenPoint{})`, `[len([1]FrozenPoint{})]int` | Use `const n = 1`, `[1]int`, or `n := len(points)` outside type syntax. |
| `go read(p)` or `defer read(p)` | Call `read(p)` synchronously. |
