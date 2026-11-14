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

type SharedPoint struct { im.Shared; X int }
type LocalRefPoint struct { im.LocalRef; X int }
type SharedLink struct { im.Shared; P *SharedPoint }
```

| Marker | Pointer policy |
| --- | --- |
| `im.Shared` | **Shareable pointers:** copy, store, pass to direct typed functions, or return them. |
| `im.LocalRef` | **Local pointers:** keep them in local variables or pass them to direct typed non-retaining functions. Value copies can be stored and returned. |

In Rust terms, `Shared` resembles `Rc<T>`/`Arc<T>` with Go's GC managing lifetime;
`LocalRef` resembles a scoped `&'a T`.

The allowed stored types are complete below. Every field, key, and element
must recursively match a row.

| Allowed form | Example |
| --- | --- |
| Boolean, integer, float, complex, or string | `bool`, `int`, `uintptr`, `float64`, `complex128`, `string` |
| Defined scalar | `type Count int` |
| Unnamed array of allowed elements | `[2]int`, `[2]SharedPoint` |
| Struct with one embedded marker and allowed fields | `SharedPoint`, `LocalRefPoint` above |
| Embedded marker alias | `type Mark = im.Shared; type Value struct { Mark; X int }` |
| Nested marked value, with either marker on the parent or child | `type LocalParent struct { im.LocalRef; Child LocalRefPoint }`, `type SharedParent struct { im.Shared; Child LocalRefPoint }` |
| Pointer to a concrete Shared struct | `*SharedPoint` |
| Recursive marked struct | `type SharedNode struct { im.Shared; Next *SharedNode }` |
| Frozen collection of allowed types | `im.FrozenSlice[SharedPoint]`, `im.FrozenMap[string, *SharedPoint]` |
| Alias or defined marked struct | `type Position = SharedPoint`, `type Position SharedPoint` |

Rejected type examples (**not exhaustive**):

| Rejected form | Replacement |
| --- | --- |
| `type Bad struct { Mark im.Shared }` | `type Good struct { im.Shared }` |
| `type Bad struct { *im.Shared }` | Embed `im.Shared` by value. |
| `type Bad struct { im.Shared; im.LocalRef }` | Embed one marker. |
| `type Bad struct { Child SharedPoint }` | `type Good struct { im.Shared; Child SharedPoint }` |
| `type Bad struct { Children []SharedPoint }` | `type Good struct { im.Shared; Children im.FrozenSlice[SharedPoint] }` |
| `type Bad struct { im.Shared; P *LocalRefPoint }` | Store `LocalRefPoint` by value, or use `*SharedPoint` for sharing. |
| `type Bad struct { im.LocalRef; P *LocalRefPoint }` | `type Good struct { im.LocalRef; P LocalRefPoint }` |
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

## Construction and pointers

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

### Values and scalar reads

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

### Pointers: both markers

These forms apply to both `*SharedPoint` (`im.Shared`) and
`*LocalRefPoint` (`im.LocalRef`).

| Allowed form | Example |
| --- | --- |
| Declare a zero or nil pointer | `var p *SharedPoint`, `var p *LocalRefPoint = nil` |
| Construct fresh storage | `p := &SharedPoint{X: 1}`, `p := new(SharedPoint)` |
| Copy a value into fresh storage | `p := new(point)`, `p := new(SharedPoint{X: 1})` |
| Copy, rebind, or discard a pointer | `q := p`, `p = q`, `p = nil`, `_ = p` |
| Read a field or copy the whole value | `x := p.X`, `value := *p` |
| Compare pointers or nil | `same := p == q`, `if p != nil { ... }` |
| Pass to a direct function with an exact concrete parameter type | `read(p)`; `func read(p *SharedPoint) int { return p.X }` |

### Shareable pointers: `im.Shared` only

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

## Atomic LocalRef slots

`im.Owned[T]` holds the current pointer to a concrete, valid `im.LocalRef`
struct. The slot is mutable; the object is immutable. `Load` returns nil
before the first `Store`. `Store` and `Load` use atomic pointer operations.
An earlier loaded pointer remains valid after a later `Store`. Go does not
specify when an unused object is collected.

These allowed forms are complete. Other slot and holder operations are rejected.
Parentheses preserve these forms.

| Allowed form | Example |
| --- | --- |
| Named slot field in an unmarked struct | `type Loader struct { Current im.Owned[LocalRefPoint] }` |
| Zero package slot | `var current im.Owned[LocalRefPoint]` |
| Zero holder local or package variable | `var loader Loader` |
| Fresh holder pointer in a local | `loader := &Loader{}` or `loader := new(Loader)` |
| Copy a holder pointer to one local | `other := loader` |
| Pass a holder pointer to a direct, concrete function | `publish(loader)` where `func publish(l *Loader)` |
| Store a fresh object in a direct slot field or package slot | `loader.Current.Store(&LocalRefPoint{X: 1})`, `current.Store(new(LocalRefPoint))` |
| Load into one local pointer | `point := loader.Current.Load()`, `var point = current.Load()`, `point = current.Load()` |

A named function can publish in a background goroutine:

```go
func publish(l *Loader) { l.Current.Store(&LocalRefPoint{X: 1}) }
go publish(loader)
```

A holder is a struct with a direct named slot field. Use holder pointers for
calls; do not copy a holder or slot value. Slot and holder pointer captures,
interfaces, indirect calls, user generic arguments, and `go` or `defer` slot
operations are unsupported. A loaded pointer follows all LocalRef pointer
rules. Load it into a local before use. A Store argument must be the direct
`&T{...}` or `new(T)` construction for the slot's exact `T`.

Rejected examples are not exhaustive:

| Rejected form | Allowed replacement |
| --- | --- |
| `im.Owned[SharedPoint]` | Use `im.Owned[LocalRefPoint]`. |
| `copy := loader.Current` or `copy := *loader` | Keep and pass `loader` as a pointer. |
| `loader.Current.Store(point)` | `loader.Current.Store(&LocalRefPoint{X: point.X})` |
| `return loader.Current.Load()` | `point := loader.Current.Load(); return *point` with a value result. |
| `read := loader.Current.Load` | `point := loader.Current.Load()` |
| `loader.Current.Load().X = 2` | `loader.Current.Store(&LocalRefPoint{X: 2})` |
| `type Outer struct { Value Loader }` | Keep the slot field in the unmarked holder. |

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
concretely typed; dependencies are checked.

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
| Declare from make, literal, zero, or Clone | `values := make([]int, 2)`, `values := []int{1}`, `var values []int`, `values := frozen.Clone()` |
| Initialize with bare `nil` | `var values []int = nil`, `var scores map[string]int = nil` |
| Read or replace a slot | `x := values[i]`, `values[i] = x`, `scores[key] = x` |
| Read length or slice capacity | `len(values)`, `cap(values)` |
| Append individual elements to the same variable | `values = append(values, 1, 2)` |
| Delete, clear, or copy | `delete(scores, key)`, `clear(values)`, `copy(destination, source)` |
| Range over elements | `for i, value := range values { ... }` |
| Consume with Freeze | `frozen := im.FreezeSlice(values)`, `frozen := im.FreezeMap(scores)` |

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

A branch or loop cannot make a consumed handle usable again. Ordinary scalar
clones that are never frozen follow ordinary Go container rules.

### Reads and Clone

Zero frozen values are empty: `var values im.FrozenSlice[int]` or
`values := im.FrozenSlice[int]{}`; the same forms work for `im.FrozenMap`.

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
