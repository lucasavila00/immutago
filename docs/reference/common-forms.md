# Common source forms

These forms apply to:

- Structs that embed `im.Frozen` or `im.Local`.
- Named interfaces that embed [im.FrozenInterface](frozen-interfaces.md).
- `im.FrozenString`, `im.FrozenSlice`, `im.FrozenMap`, and local read projections.

Combine them with each type's forms. [Cell](cell.md) has its own forms.
Ordinary [Copied values](copied.md) and their ordinary holders use all Go forms.
Ordinary [Frozen pointer handles and carriers](frozen-records.md#stored-pointers--exhaustive)
also use Go storage and transport rules. Their selected Frozen objects keep the rules below.
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
| User generics with ordinary or persistent Frozen parameters, results, constraints, receivers, and type arguments | `identity(text)`, `identity[im.FrozenString](text)`, `func keep[T FrozenPoint](value T) T { return value }` |
| Generic function and method bodies | `func read[T any](value T) int { text := im.FreezeString("x"); return text.Len() }`; each operation keeps its normal source rules. |
| Ordinary generic types | `type Plain[T any] struct { Value T }`, used as `Plain[int]`, `Plain[*FrozenPoint]`, or `Plain[*Cache]` for a Cell holder; instantiated fields still need their normal field proofs. |

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `type Box[T any] struct { im.Frozen; Value T }` | `type Box struct { im.Frozen; Value FrozenPoint }` |
| `Plain[FrozenPoint]` stores a Frozen field in an unmarked record | Use a concrete Frozen record for this stored field. |
| `identity(local)` with a Local record or projection | Copy it directly, or use a direct function with a concrete parameter. |
| `func use[T LocalPoint](value T)` | Use `func use(value LocalPoint)`. |
| `T{point}` with `T ~[]any` | Construct `[]any{point}` first, then convert it to T. |
| Convert a type parameter to a Frozen interface inside a generic body | Convert the concrete implementation to that Frozen interface before the generic call. |

## Use Frozen values

Frozen records, FrozenString, FrozenSlice, FrozenMap, Frozen interfaces, and
unnamed arrays of these values can persist. Aliases keep these forms. Pointers
use the separate [pointer forms](#common-pointer-forms). Local values and
projections do not use this table.

| Allowed forms — exhaustive | Example |
| --- | --- |
| Put a value in an ordinary interface by assignment, argument, result, comparison, or explicit conversion | `var boxed any = text`, `useAny(text)`, `return text`, `boxed == text`, `any(text)` |
| Use ordinary interface fields and collection entries | `struct{ Value any }{text}`, `[]any{text}`, `map[any]any{point: text}` |
| Recover a concrete Frozen value with an assertion or type switch | `text := boxed.(im.FrozenString)`, `text, ok := boxed.(im.FrozenString)`, `switch value := boxed.(type) { case FrozenPoint: use(value) }` |
| Use an ordinary function or interface method with persistent inputs and results | `callback(text)`, `factory.Read()`, `value, ok := callback()` |
| Use a user generic function, with inferred or explicit type arguments | `identity(text)`, `identity[im.FrozenString](text)` |
| Capture local or package values in closures | `read := func() int { return text.Len() }` |
| Select a method value or method expression with ordinary or persistent inputs and results | `read := text.Len`, `read := FrozenPoint.XValue`, `read := Scorer.Score` |
| Bind or replace several values, including callback results | `a, b := point, other`, `a, b = b, a`, `var a, b = callback()` |
| Call with go or defer when operands use these persistent forms | `go use(text)`, `defer text.Len()` |
| Save or call an iterator that yields persistent values | `items := frozen.All()`, `for value := range items { use(value) }` |

Each field and collection argument still needs its type proof. Fields inside a
Frozen record stay read-only, including inside a closure or generic body. A
conversion to a Frozen interface still proves the concrete implementation and
its exact method receivers. An assertion from `any` to a Frozen interface cannot
supply that proof; recover a proved concrete record first.

Generic helpers use ordinary Go contracts. Do not pass a Frozen pointer to a
helper that changes its target.

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
- Frozen values use the forms above. Local values and pointers keep their restricted forms.
- Constant expressions follow the same rules. For a Local array, compute
  `count := len(points)` before a closure, then capture `count`. Frozen arrays can be captured.

## Common pointer forms

The construction and target-read forms apply to both `*FrozenPoint` and
`*LocalPoint`. Frozen pointer handles use ordinary Go storage and transport.
Local pointers use only the direct-call forms below and cannot escape.

| Allowed forms — exhaustive | Example |
| --- | --- |
| Declare a zero or nil pointer | `var p *FrozenPoint`, `var p *LocalPoint = nil` |
| Create a new object | `p := &FrozenPoint{X: 1}`, `p := new(FrozenPoint)` |
| Create a new object from a value | `p := new(point)`, `p := new(FrozenPoint{X: 1})` |
| Keep a selected Frozen root | `p := cell.Load().Unsafe()` |
| Copy, replace, or discard a pointer | `q := p`, `p = q`, `p = nil`, `_ = p` |
| Read a field or copy the whole value | `x := p.X`, `value := *p` |
| Compare pointers or nil | `same := p == q`, `if p != nil { ... }` |
| Pass a Local pointer to a direct function with an exact concrete parameter type | `read(p)`; `func read(p *LocalPoint) int { return p.X }` |
| Call a Local pointer method with the exact receiver type | `p.Read()`; `func (p *LocalPoint) Read() int { return p.X }` |

- Type aliases and parentheses preserve these forms.
- Pointer targets must be structs with `im.Frozen` or `im.Local`.
- Construct pointers with the listed `new` or `&T{...}` forms.
  A generated record projection can also supply its selected pointer through Unsafe.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `point.X = 2`, `p.X = 2`, `*p = FrozenPoint{}` | `point = FrozenPoint{X: 2}` or `p = &FrozenPoint{X: 2}` |
| `p := &point` | `p := new(point)` |
| `&point.X` | Read or copy the scalar. |
| `new(im.FrozenSlice[int])` | Pass the frozen wrapper by value. |
| `&[2]FrozenPoint{}` | Put the array in a marked struct. |
| `(*LocalPoint)(nil)` | `var p *LocalPoint = nil` |

## Methods

Methods on protected value receivers and Local pointers use these forms.
Methods on Frozen pointer handles use ordinary Go calls; their bodies still
cannot change Frozen objects.

```go
func (p FrozenPoint) XValue() int { return p.X }
func (p *FrozenPoint) ReadX() int { return p.X }

type PointGroup struct { im.Frozen; FrozenPoint FrozenPoint }

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
| Ordinary concrete receiver with Local inputs or results | Direct, non-generic calls. Local inputs and results keep their im rules. Persistent values use the forms above. |
| Fixed library type | Listed API forms. |

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `point.ReadX()` adds an implicit address | Construct `p := &FrozenPoint{X: 3}`, then call `p.ReadX()`. |
| `p.XValue()` adds an implicit dereference for a Local pointer | `(*p).XValue()` |
| `type PointGroup struct { im.Frozen; FrozenPoint }` embeds data | Give the field a name, then call `group.FrozenPoint.XValue()`. |
| `p.Counter.Inc()` calls a method on a protected scalar field | Copy the scalar first: `count := p.Counter; count.Inc()`. |
| `read := local.ReadX` or `(*LocalPoint).ReadX(local)` | `local.ReadX()` |
| `go local.ReadX()` or `defer local.ReadX()` | `local.ReadX()` |

## Statements and calls

Use the operations above inside these statement forms:

| Allowed forms — exhaustive | Example |
| --- | --- |
| Declare one variable or assign one destination | `var value FrozenPoint`, `value := FrozenPoint{X: 1}`, `value = FrozenPoint{X: 2}` |
| Change a native builder slot or its Copied fields | `values[0] = value`, `counts[0]++`, `counts[0] += 2`, `scores[0].Value++` |
| Bind several results from one direct named function, concrete method, or proved Frozen interface method; check each destination separately | `value, found, out := reader.Lookup(key, buf)`, `value, state.Found, out = read()`, `var value, found, out = read()` |
| Freeze a slice or map in its own assignment or declaration | `frozen := im.FreezeSlice(values)`, `_ = im.FreezeSlice(values)` |
| Call a direct named function | `read(p)`, `n := read(p)` |
| Call a method in the list above | `n := p.ReadX()` |
| Return values or Frozen pointers, including multiple results | `return point`, `return p, true` |
| Return named results, or forward call results to a return or direct typed call | `func read() (value LocalPoint, found bool) { value, found = makePair(); return }`, `return makePair()`, `takePair(makePair())` |
| Conditional or value switch | `if ready { p = makeFrozenPoint() }`, `switch p.X { ... }` |
| Condition, three-clause, or unconditional loop | `for p.X > 0 { ... }`, `for i := 0; i < p.X; i++ { ... }`, `for { ... }` |
| Range with fresh loop variables | `for i, value := range values { ... }`, `for i := range count { ... }` |
| Assign one existing writable loop target, using the assignment rules above | `for i = range count { ... }`, `for a[point.X] = range a { ... }` |
| Use checked operations in a select case body | `case <-ready: values[0] = value` |

- Local value and builder-slot assignments have one destination, except for the
  result-call forms above. Local-pointer and builder-handle bindings have one
  destination. Local pointers cannot be returned. Frozen values also use the forms above.
  Returns can have several results.
- Fixed calls such as FrozenMap.Get, MapProjection.Get, and supervise.Start use
  the same result binding forms. For example, `value, found = frozen.Get(key)`
  can reuse local names, and `var ref, err = supervise.Start[Worker](root, args)`
  can declare the results.
- Multiple-result bindings with Local values or builder-slot operations require
  one direct typed call that supplies all results.
  Each result keeps its type, write, and storage checks.
  The `:=`, `=`, and `var` forms are allowed, including reused locals in `:=`.
  Each Local result needs a local name or blank. Ordinary results can use local
  or package names, ordinary fields, elements, or pointer targets. A native builder
  slot can receive an allowed element; the builder handle cannot come from a tuple.
  Aliases and parentheses keep these forms. For example, `local, found := makeLocalPair()`
  can bind a Local record, and `local, state.Found = makeLocalPair()` can store the bool.
  `var _, found = makeLocalPair()` can discard the Local result at package scope.
  An assignment cannot store a Local value in persistent storage or change a protected
  field. Persistent Frozen results can be boxed in `any`.
- Each [service result](service-results.md) must be a read result or the predeclared
  error type. There is no result count or order limit. Calls with only ordinary
  values keep Go rules. Tuple callbacks and generics keep each component's normal
  rules: persistent Frozen values are allowed, while Local values need direct typed calls.
- Blocks, `break`, and `continue` follow ordinary Go rules.
- Local values and pointers need the exact parameter type, such as
  `func read(p *LocalPoint) int`. Call directly, without `go` or `defer`.

| Multiple-result destinations — exhaustive | Example |
| --- | --- |
| Local names or blanks | `view, found := readPair()`, `var view, found = readPair()`, `view, _ = readPair()` |
| Package names for ordinary results | `view, savedFound = readPair()`, `var _, savedFound = readPair()` |
| Ordinary fields | `view, state.Found = readPair()` |
| Ordinary array, slice, or map elements | `view, foundByKey[key] = readPair()` |
| Ordinary pointer targets | `view, *found = readPair()` |
| Native builder slots with allowed element types | `points[0], view = readPoint()` |

The normal write and type rules apply to every destination. Unlisted restricted
binding forms are rejected.

| Rejected examples — non-exhaustive | Replacement |
| --- | --- |
| `point.X, found = readPair()` when point is Frozen | Bind local names, then construct a new point. |
| `values, found := readSlice(); _ = im.FreezeSlice(values)` | Construct a fresh builder or Clone first, then Freeze it. |
| `value, p = LocalPoint{}, &LocalPoint{}` | Use two statements. |
| `Other(point)` where `type Other FrozenPoint` | Construct `Other{X: point.X}`. |
| `OtherFrozen(frozen)` where `type OtherFrozen im.FrozenSlice[int]` | Keep the frozen type; use `Clone` for mutable storage. |
| `for a[point.X], n = range values { ... }` | Bind fresh loop locals, then assign each destination separately. |
| `use(im.FreezeSlice(values))` | Bind Freeze to a local, then call `use`. |
| `case _, values[0] = <-ready:` | Receive first; write in the case body. |
| `case <-watch(im.FreezeSlice(values)):` | Freeze and compute the channel before the select; receive from the channel variable. |
| `var value any = local` for a Local pointer | Use a concrete `*LocalPoint` variable. |
| `callback(local)` through a function variable with a Local pointer | Use a direct named function. |
| `callback := func() int { return local.X }` with LocalPoint | `x := local.X; callback := func() int { return x }` |
| `func() int { return len([1]*LocalPoint{p}) }` | Use `1`, or compute an ordinary length before the closure. |
| `const n = len([1]FrozenPoint{})`, `[len([1]FrozenPoint{})]int` | Use `const n = 1`, `[1]int`, or `n := len(points)` outside type syntax. |
| `go read(local)` or `defer read(local)` for a Local pointer | Call `read(local)` synchronously. |
