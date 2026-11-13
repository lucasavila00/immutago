# Language rules

[Documentation](README.md) · [Get started](getting-started.md) · [Collection API](../pkg/im/readme.md)

Marked objects are immutable from construction. You can copy and share them.
Go's garbage collector manages their lifetime. A variable can be rebound to a
new value, but a field, element of protected storage, or pointer target cannot
be changed. Prepare scalar locals first, then construct the final object.

The allowed Go forms in this guide define the complete supported language.
The checker must match each complete form and prove its type rules. Unlisted
operations are rejected. Rejection examples are not exhaustive.

## Choose a marker

Both markers make an object immutable from construction. Neither permits
mutable initialization, field writes, or writes through a pointer.

| Marker | Complete reference policy |
| --- | --- |
| `im.Immut` | References can be local, copied, rebound, stored, passed to direct typed functions, returned, and captured by a closure. |
| `im.ImmutNoRef` | References can be local, copied, rebound, and passed to direct typed functions that do not retain them. References cannot be stored, returned, or captured. |

Embed exactly one selected marker by value. A struct with both markers is rejected. `ImmutNoRef` restricts references, not value
copies: a value copy can be stored or returned. Go manages object lifetime for
both markers. Neither marker supplies resource destruction or a general memory
leak guarantee.

```go
type Point struct { im.Immut; X int }
type Local struct { im.ImmutNoRef; X int }
func readLocal(value *Local) int { return value.X }
func copyLocal(value Local) Local { return value }
```

The reference table below describes `Immut`. The complete additional rules for
`ImmutNoRef` appear in [local-only references](#local-only-references).

## Find a rule

- [Value types](#allowed-value-types): which fields and collection arguments pass.
- [Objects](#immutable-objects) and [references](#shared-references): construction and sharing.
- [Calls](#constructors-and-calls): constructors and readers.
- [References](#what-you-can-do-with-a-reference): complete allowed operations.
- [Statements](#what-statements-you-can-use): complete allowed statement forms.
- [Collections](#frozen-collections): Freeze, builders, reads, and Clone.
- [Scope](#what-we-do-not-implement): checks and unsupported Rust features.

Here, **protected storage** means marked values, their fields, frozen
collections, and arrays containing protected values. A **builder** is a native map or slice checked for exclusive
ownership. The builder section states which native containers are checked.
Examples with `...` in a block omit ordinary code; the ellipsis is not Go syntax.


## Allowed value types

A stored field or collection argument must match one of these forms. Each
component is checked recursively; all fields, keys, and elements must pass.

| Allowed type | Go example |
| --- | --- |
| Boolean, integer, floating-point, complex, or string scalar | `bool`, `int`, `uintptr`, `float64`, `complex128`, `string` |
| A defined scalar type | `type Count int` |
| An unnamed array of allowed elements | `[2]int` or `[2]Point` |
| A struct that directly embeds `im.Immut` or `im.ImmutNoRef` by value | `type Point struct { im.Immut; X, Y int }` |
| A pointer to a recursively protected immutable type | `*Point`, `*im.FrozenSlice[Point]`, `*[2]Point` |
| A frozen collection of allowed types | `im.FrozenSlice[Point]` or `im.FrozenMap[string, Point]` |
| A type alias for an allowed type | `type Position = Point` |
| A defined struct whose underlying struct has the marker and allowed fields | `type Position Point` |
| A type parameter whose constraint proves the same rules | `type Box[T ~int | ~string] struct { im.Immut; Value T }` |

The Go examples below use this type and import:

```go
import "github.com/lucasavila00/immutago/pkg/im"

type Point struct {
    im.Immut
    X, Y int
}
```

Aliases are resolved before checking the marker and collection identities.
An alias for either marker can be embedded. A named marker field or `*im.Immut`
does not mark a struct. Every nested struct must have its own marker.

The following rejected examples are not exhaustive:

```go
// Rejected as a stored field type: a named array has no marker.
type Counts [2]int

// Allowed: wrap the array in a marked struct.
type Counts struct {
    im.Immut
    Values [2]int
}
```

Pointers to scalars and native slices, maps, channels, interfaces, and functions
cannot be fields of a marked struct. A stored pointer target must itself contain protected storage, pass the
recursive immutable type proof, and not be a direct `ImmutNoRef` target.
Inline `ImmutNoRef` values can be fields and collection elements. Arrays and
frozen wrappers with those values can have shared pointers because their
elements cannot be addressed. Unmarked structs cannot contain marked values or
frozen collections, including through arrays or native containers. Recursive
collection types are allowed when every reachable field passes:

```go
type Tree struct {
    im.Immut
    Value int
    Children im.FrozenSlice[Tree]
}
```

### Generic fields

A constraint is checked before instantiation. Every member of a union must
pass. For intersected bounds, one bound that proves immutability is sufficient.
Embedding a named constraint preserves its rules:

```go
type Scalar interface { ~int | ~string }
type Key interface { Scalar; comparable }
type Box[T Key] struct {
    im.Immut
    Value T
}
type Values[T Scalar] = im.FrozenSlice[T]
```

`any`, `comparable` alone, and method-only constraints do not prove immutable
fields. A `~[2]int` constraint is also rejected: it admits named arrays without
a marker. Use scalar bounds or a concrete marked wrapper. These rejected
examples are not exhaustive.

An unmarked generic struct must prove that its fields cannot contain protected
storage. `type Plain[T ~int] struct { Value T }` passes;
`type Plain[T any] struct { Value T }` fails.

Generic type declarations do not permit generic borrowing helpers. The
[reference rules](#what-you-can-do-with-a-reference) still require concrete
reader parameters. Generic functions used with scalar types can construct
frozen collections through the fixed `im` APIs. User generic calls cannot use
protected type arguments, even when they pass values rather than references:

```go
// Rejected: Point is a protected type argument.
copy := identity(point) // identity[T any](value T) T

// Allowed: copy directly, or use a function with concrete types.
copy := point
// func copyPoint(value Point) Point { return value }
```

## Immutable objects

Construct the complete value after conditional scalar work:

```go
x := 20
if condition {
    x = 10
}
point := Point{X: x, Y: 40}
point = Point{X: 30, Y: 40} // Rebind the whole variable.
```

A field write is rejected even before sharing and even after the last read:

```go
// Rejected: marked fields are immutable from construction.
point.X = 99

// Allowed: construct a new value and rebind the variable.
point = Point{X: 99, Y: point.Y}
```

Go copies structs and arrays on assignment. A copy is also immutable. Frozen
collection fields share read-only storage. Copying does not clone collections.

## Shared references

A protected reference permits reads and sharing. It does not permit writes.
Unlike a Rust borrow, it has no checked lexical or non-lexical lifetime. Go's
garbage collector keeps a reachable object alive.

```go
point := &Point{X: 1}
other := point
point = &Point{X: 2}
x := other.X // Still 1.
```

A reference must start with fresh construction: `&Point{...}`, `new(Point)`,
or `new(value)`. The expression form of `new` copies the value into new storage.
Taking `&point` or `&shape.Child` is rejected. Rebinding an addressed variable
would change an object that other references observe.

References can be stored in marked fields and frozen collections when their
resolved target types pass the recursive proof:

```go
type Node struct {
    im.Immut
    Value int
    Next *Node
}
func prepend(next *Node) *Node {
    return &Node{Value: 1, Next: next}
}
```

## Constructors and calls

Named functions can construct, receive, and return protected values and shared
references. Calls must be direct and typed. Calls with `ImmutNoRef` references must be
synchronous and must not retain those references. The checker checks
the callee too, including callees in other packages.

```go
func makePoint(flag bool) *Point {
    x := 0
    if flag {
        x = 1
    }
    return &Point{X: x}
}
func read(point *Point) int { return point.X }
```

A callback with only scalar parameters and results can receive scalar copies.
A closure can capture an `Immut` reference or an immutable value copy of either
mode. It cannot capture an `ImmutNoRef` reference. Copy a scalar first when the reference must stay local:

```go
x := point.X
callback := func() int { return x }
```

## What you can do with a reference

This list is **exhaustive** for references to protected storage. A target can
be an `Immut` struct, frozen collection, or array containing protected values.
A direct pointer to `ImmutNoRef` uses the local-only rules below.
Scalar fields and scalar-only arrays cannot be addressed through protected
storage. Unlisted operations are rejected.

| Allowed form | Go example |
| --- | --- |
| Declare a zero reference | `var p *Point` |
| Construct an immutable object | `p := &Point{X: 1}` or `p := new(Point)` |
| Construct from a value copy | `p := new(point)` or `p := new(Point{X: 1})` |
| Construct a protected array | `p := &[2]Point{}` |
| Start or defer a direct call with shared references | `go read(p)` or `defer read(p)` |
| Capture an `Immut` reference | `callback := func() int { return p.X }` |
| Copy or rebind a reference variable | `q := p` or `p = q` or `p = nil` |
| Store in a marked field | `node := Node{Next: p}` |
| Store in an exclusive builder | `values := []*Point{p}` |
| Store in a package variable | `saved = p` |
| Return a reference | `return p` or `return &Point{X: 1}` |
| Discard a reference | `_ = p` |
| Read a field | `x := p.X` |
| Read an explicitly dereferenced array | `x := (*p)[0].X` |
| Call a listed frozen method | `n := p.Len()` |
| Copy the complete value | `next := *p` |
| Compare references or nil | `same := p == q` or `if p != nil { ... }` |
| Pass to a direct named function | `x := read(p)` or `x := read(&Point{X: 1})` |

Parentheses and Go type aliases preserve these forms. Concrete pointer
parameters and results are allowed. User-defined methods on protected values,
including promoted methods, are unsupported. Only the listed `im` methods
`Get`, `At`, `Len`, `All`, and `Clone` are supported.

## Local-only references

For `ImmutNoRef`, this is the complete allowed reference list. Unlisted forms
are rejected. A referenced type must pass the same recursive immutable proof.

| Allowed form | Go example |
| --- | --- |
| Declare a zero local reference | `var p *Local` |
| Construct a fresh local object | `p := &Local{X: 1}` or `p := new(Local)` |
| Construct from a value copy | `p := new(local)` or `p := new(Local{X: 1})` |
| Copy or rebind a local reference | `q := p` or `p = q` or `p = nil` |
| Read or discard a local reference | `x := p.X` or `_ = p` |
| Copy the referenced value | `value := *p` |
| Compare references or nil | `same := p == q` or `if p != nil { ... }` |
| Pass to a direct typed non-retaining function | `x := readLocal(p)` |
| Declare a concrete reader parameter | `func readLocal(p *Local) int { return p.X }` |

Value copies remain ordinary immutable values:

```go
func localValue() Local { return Local{X: 1} }
type Snapshot struct { im.Immut; Value Local }
```

The following rejected examples are not exhaustive:

```go
// Rejected: a local-only reference leaves the function.
func escape() *Local { return &Local{X: 1} }

// Allowed: return a value copy.
func copyOut() Local { return Local{X: 1} }
```

```go
// Rejected: storing a local-only reference.
type Saved struct { im.Immut; Value *Local }

// Allowed: store the immutable value.
type SavedValue struct { im.Immut; Value Local }
```

References to `ImmutNoRef` also cannot enter a package variable, native builder,
frozen collection, closure, interface, deferred call, or goroutine. The checker
checks reader bodies and dependencies; a concrete parameter does not grant
permission to retain the pointer.

## What you cannot do with a reference

These examples are **not exhaustive**. The allowed list above is complete.

| Rejected Go example | Allowed replacement |
| --- | --- |
| `p.X = 1` or `*p = Point{}` | `p = &Point{X: 1}` |
| `value.Read()` | `Read(&Point{X: value.X})` |
| `var x any = p` | Use a concrete `*Point` variable. |
| `func() { readLocal(localPointer) }` | Call `readLocal(localPointer)` directly. |
| `defer readLocal(p)` or `go readLocal(p)` | Call `readLocal(p)` synchronously. |
| `p[0].X` | `(*p)[0].X` |
| `&point` or `&shape.Child` | Construct `&Point{X: point.X}`. |
| `&point.X` | Read `point.X`, or copy it to a scalar local. |
| `&p` | Pass `p` directly. |
| `readGeneric(p)` | Use a concrete `*Point` parameter. |

The checker enforces these rules. There are no caller obligations or escape
hatches that permit rejected operations.

## What statements you can use

This list is **complete** for statements that use protected storage or a builder.
Use the reference and builder operations listed in this guide within these forms.

| Allowed form | Go example |
| --- | --- |
| Declare one value or builder | `value := Point{X: 1}` or `values := []int{1, 2}` |
| Declare one shared reference | `p := &Point{X: value.X}` |
| Assign one destination | `value = Point{X: 1}` or `values[0] = 2` |
| Update one destination | `values[0]++` or `values[0] += 2` |
| Bind both results of the listed map reader to new locals | `value, found := frozen.Get(key)` |
| Freeze into one variable | `frozen := im.FreezeSlice(values)` |
| Freeze and discard the result | `_ = im.FreezeSlice(values)` |
| Start or defer a direct call with `Immut` references | `go read(p)` or `defer read(p)` |
| Call a named reader | `read(p)` or `n := read(&Point{X: 1})` |
| Return one or more values | `return value` or `return value, found` |
| Forward a named function's results directly | `return makePair()` or `takePair(makePair())` |
| Use ordinary value switches | `switch value.X { case 1: value = Point{X: 2} }` |
| Use `if` and ordinary loops | `if ready { value = Point{X: 1} }` or `for i := range values { values[i] = i }` |
| Assign one loop target | `for i = range count { ... }` |
| Use checked operations inside a `select` case body | `case <-ready: values[0] = true` |

Writes can rebind complete variables or change native builder storage.
Writes to marked fields, protected array elements, or pointer targets fail.
A consumed builder cannot be used again. Blocks, `break`, `continue`, and other
control flow without protected operands follow ordinary Go rules. A Freeze call must be the only
right-hand-side expression of its assignment or declaration. The two-result
`Get` form creates new locals; it cannot assign into existing protected storage.
The one-destination limit applies to assignments and declarations, not to
returns. Multiple results may contain protected values or `Immut` shared references. A named function can pass those results directly to another named
function with matching concrete parameters:

```go
func makePair() (Point, bool) { return Point{X: 1}, true }
func forwardPair() (Point, bool) { return makePair() }
func takePair(point Point, found bool) { ... }

// Allowed inside a function: no local multiple assignment.
takePair(makePair())

// Rejected: two local destinations with a protected result.
point, found := makePair()
```

Use a marked result struct when callers need to bind several related results
together. Its declaration has one destination.

## Statement examples that are not allowed

These examples are **not a complete list**. Only the forms above are allowed.

```go
// Rejected: two destinations involving protected storage.
value, p = Point{X: 1}, &Point{}

// Allowed: one destination per statement.
value = Point{X: 1}
p := &Point{X: value.X}
```

```go
// Rejected: two range targets, including a protected read in an index.
for a[value.X], n = range numbers { ... }

// Allowed: bind new loop variables, then assign each destination.
for i, number := range numbers {
    a[value.X] = i
    n = number
}
```

```go
// Rejected: Freeze inside another expression.
use(im.FreezeSlice(values))
return im.FreezeSlice(values)

// Allowed: bind the frozen value first.
frozen := im.FreezeSlice(values)
use(frozen)
return frozen
```

```go
// Rejected: a builder in a select receive target or channel expression.
select {
case _, values[0] = <-ready:
case <-watch(im.FreezeSlice(values)):
}

// Allowed: receive first, then write inside the case body.
select {
case <-ready:
    values[0] = true
}
frozen := im.FreezeSlice(values)
```

## Frozen collections

Build a native Go container, then transfer its storage to a frozen view:

```go
scores := make(map[string]int, 2)
scores["alice"] = 10
scores["bob"] = 20
frozen := im.FreezeMap(scores)
score, found := frozen.Get("alice")
// scores["alice"] = 30 // rejected: scores has been consumed
```

`im.FreezeSlice([]int{1, 2})` works the same way. The resulting types are
`im.FrozenMap[K, V]` and `im.FrozenSlice[T]`. They have no update methods.
Freeze does not copy or allocate. See the [zero-overhead target](performance.md#zero-abstraction-overhead)
for the performance contract.

In Rust terms, freezing **moves** the builder into a read-only owner. Go does not
perform that move itself: the checker rejects every later use of the original
handle. Copies of the frozen wrapper share read-only storage and remain valid.
Keys and elements must recursively satisfy the immutable type rules above.

For an update, clone into independent mutable storage, edit it, and freeze again:

```go
next := frozen.Clone()
next["alice"] = 30
updated := im.FreezeMap(next)
// frozen still contains alice: 10; updated contains alice: 30.
```

`Clone` costs O(n) time and storage. It is a shallow copy; recursively immutable
elements make that sufficient. Scalar clones that you do not freeze can be used
as ordinary mutable Go containers. If you later freeze a clone, its earlier
uses must satisfy the exclusive-builder rules too. This API has no persistent
path-copy updates.

### What you can do with a builder

This list is **exhaustive** for a native container that you freeze, and for native
containers whose keys or elements contain marked values or frozen collections.
Unlisted operations are rejected.

| Allowed form | Go example |
| --- | --- |
| Allocate a local slice or map | `values := make([]int, 2)` or `scores := make(map[string]int)` |
| Initialize a local from a native literal | `values := []int{1, 2}` or `scores := map[string]int{"alice": 10}` |
| Declare a zero or nil container | `var values []int` or `var scores map[string]int = nil` |
| Clone a frozen collection into a local | `values := frozen.Clone()` |
| Read an element | `value := values[i]` or `score := scores[key]` |
| Write an element | `values[i] = 2` or `scores[key] = 10` |
| Read the length or slice capacity | `n := len(values)` or `n := cap(values)` |
| Append values to the same slice variable | `values = append(values, 1, 2)` |
| Delete a map entry | `delete(scores, key)` |
| Clear a container | `clear(values)` or `clear(scores)` |
| Copy slice elements | `copy(destination, source)` |
| Read elements in a loop | `for i, value := range values { ... }` |
| Freeze into one variable | `frozen := im.FreezeSlice(values)` or `frozen := im.FreezeMap(scores)` |

A local declaration can use `var` instead of `:=`. Each source and destination
builder in `copy` must satisfy these rules. `append` takes individual elements;
it does not accept another slice with `...`.

These direct fresh inputs are also allowed. The outer Freeze must still have
its own assignment or declaration:

| Fresh input | Go example |
| --- | --- |
| Native allocation | `frozen := im.FreezeSlice(make([]int, 2))` |
| Native literal | `frozen := im.FreezeMap(map[string]int{"alice": 10})` |
| Nil | `frozen := im.FreezeSlice[int](nil)` |
| Direct Clone result | `next := im.FreezeMap(frozen.Clone())` |

Each form applies to both matching collection types. A builder cannot
be frozen while a range over it is active. Branches and loops cannot make a
consumed handle usable again.

### What you cannot do with a builder

This list is **not exhaustive**; the list above defines what is supported.

| Rejected example | Allowed replacement |
| --- | --- |
| `alias := values` | `frozen := im.FreezeSlice(values)` shares read-only storage. |
| `part := values[1:]` | Read elements with `values[i]`. |
| `p := &Point{X: value.X}s[0]` | Copy the element with `value := values[0]`. |
| `update(values)` | Update elements in the current function before Freeze. |
| `return values` | Freeze into a local, then return the frozen value. |
| `im.FreezeSlice(parameter)` | Create and fill a local builder, then freeze it. |
| `values[0] = 2` after Freeze | Use `next := frozen.Clone()`, then change `next[0]`. |

These restrictions prevent writable aliases from surviving the ownership
transfer. The checker verifies them; they are not obligations left to the caller.

### Reading frozen storage

These are the supported collection methods and direct call forms:

| Allowed form | Go example |
| --- | --- |
| Read the length | `n := frozen.Len()` |
| Read a slice element | `value := frozen.At(i)` |
| Read a map entry into new locals | `value, found := frozen.Get(key)` |
| Iterate over slice or map entries | `for key, value := range frozen.All() { ... }` |
| Clone into mutable native storage | `next := frozen.Clone()` |

The receiver can be an expression of the frozen collection type, such as
`frozen`, `value.Items`, or `makeFrozen()`, or a supported shared reference.
Parentheses are allowed: `(frozen.Len)()` and `range (frozen.All())` have the same
meaning. `Len`, `At`, and `Get` allocate no storage. `All` yields value copies
without cloning the container. `Clone` creates independent storage.
See [collection costs](performance.md).

These rejected examples are not exhaustive:

```go
// Rejected: save a method or call it through the type.
read := frozen.At
value := im.FrozenSlice[int].At(frozen, 0)

// Allowed: call the method on the collection.
value := frozen.At(0)
```

An iterator that carries marked values must be consumed by direct `range`:

```go
// Rejected when frozen contains marked values.
entries := frozen.All()

// Allowed: copy each element in a direct range.
for key, value := range frozen.All() { ... }
```

Iterators whose key and value types contain no marked values or frozen
collections follow ordinary Go iterator rules. For example, you can save
`entries := frozen.All()` when the elements are `int` or `[2]int`.

## What we do not implement

There are no mutable borrows, lifetime annotations, partial moves, or Rust
resource ownership and destruction rules. Go still supplies its own value-copy
semantics and garbage collector.

The checker enforces these rules, including across package boundaries. You do
not have to uphold extra rules by hand. If the checker cannot prove an operation
safe, it reports an error; it does not skip the operation and accept the program.
Imports of `unsafe` and `reflect` are rejected in checked code, as are
unsupported aliasing and function declarations without Go bodies. These rules
also apply when the affected function uses only scalar types. For example:

```go
// Rejected: no Go body is available to check.
func readCount() int

// Allowed: provide a Go implementation.
func readCount() int { return 1 }
```

An assembly or foreign implementation cannot be trusted through an annotation.
There are no escape hatches or suppression directives.

Package variables can be rebound as complete values. Their protected fields
and pointer targets cannot be written.

Run `immutago ./...` or `go vet -vettool=<absolute path to immutago> ./...`.
An ordinary Go build does not run the checker.

The [architecture](arch/README.md) explains the implementation. The
[source notes](../third_party/rustc/README.md) identify the rustc algorithms
used by the implementation.
