# Language rules

[Documentation](README.md) · [Get started](getting-started.md) · [Collection API](../pkg/im/readme.md)

For marked values, their fields, and `im` collections, Immutago checks this
subset of Rust's ownership rules:

- **Owned values (`T`)** can be initialized, changed, and reassigned.
- **Shared borrows (`&T`)** permit reads, never writes.
- **Non-lexical lifetimes (NLL)** end a borrow after its last use.
- **Mutable borrows (`&mut T`)** are not supported.

The Rust below explains the rules. It is not an API you need to implement.

The allowed Go forms in this guide define the supported language. The checker
must match those forms and check their ownership rules. A form that does not
match must be rejected. The examples of rejected code are not exhaustive.


## Find a rule

- [Value types](#allowed-value-types): which fields and collection arguments pass.
- [Owned values](#owned-values) and [shared borrows](#shared-borrows-and-nll): the Rust model.
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
| A struct that directly embeds `im.Marker` by value | `type Point struct { im.Marker; X, Y int }` |
| A frozen collection of allowed types | `im.FrozenSlice[Point]` or `im.FrozenMap[string, Point]` |
| A type alias for an allowed type | `type Position = Point` |
| A defined struct whose underlying struct has the marker and allowed fields | `type Position Point` |
| A type parameter whose constraint proves the same rules | `type Box[T ~int | ~string] struct { im.Marker; Value T }` |

The Go examples below use this type and import:

```go
import "github.com/lucasavila00/immutago/pkg/im"

type Point struct {
    im.Marker
    X, Y int
}
```

Aliases are resolved before checking the marker and collection identities.
An alias for `im.Marker` can be embedded. A named marker field or `*im.Marker`
does not mark a struct. Every nested struct must have its own marker.

The following rejected examples are not exhaustive:

```go
// Rejected as a stored field type: a named array has no marker.
type Counts [2]int

// Allowed: wrap the array in a marked struct.
type Counts struct {
    im.Marker
    Values [2]int
}
```

Native pointers, slices, maps, channels, interfaces, and functions cannot be
fields of a marked struct. Unmarked structs cannot contain marked values or
frozen collections, including through arrays or native containers. Recursive
collection types are allowed when every reachable field passes:

```go
type Tree struct {
    im.Marker
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
    im.Marker
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

## Owned values

An owned Go local behaves like a Rust `let mut` binding. You can build through
branches and then replace the whole value:

```go
point := Point{}
if condition {
    point.X = 10
} else {
    point.X = 20
}
point = Point{X: 30, Y: 40}
```

In Rust terms, this is `let mut point: Point`, followed by field assignments.

This permits ordinary Go construction without needing Rust's `if` expressions.
Immutability does not mean “assign each field exactly once.” It means that a
shared reference cannot observe its owner being changed.

Go structs copy on assignment:

```go
next := point
next.X = 99 // point is unchanged
```

This resembles Rust's `Copy` values, not a move of a non-`Copy` value. Struct and
array fields are independent copies. Frozen collection fields share read-only
backing storage; they are not cloned by a struct copy.

## Shared borrows and NLL

Taking a pointer to an existing value creates a shared borrow:

```go
point := Point{}
p := &point
x := p.X    // Last use of p.
point.X = 1 // Allowed: the borrow has ended.
```

The Rust equivalent uses a shared reference, not `&mut Point`:

```rust
let borrowed = &point;
let x = borrowed.x; // Last use of borrowed.
point.x = 1;        // Allowed after that use.
```

Moving the write before the last use fails:

```go
p := &point
point.X = 1 // Rejected: p still needs the owner.
x := p.X
```

Writing through the reference also fails:

```go
p := &point
p.X = 1 // Rejected: this is a shared borrow, like Rust &Point.
```

Each local reference is initialized once from an owned local or its marked
child. Do not copy the reference or assign another pointer to it. Branches and
loops still determine its last use.

A borrow protects the **whole owner**. Borrowing a marked child also prevents
writes to its parent's other fields. Unlike Rust, Immutago does not split loans
between disjoint fields or array elements.

## Constructors and calls

Construct and return values, then borrow a local value when needed:

```go
func makePoint(flag bool) Point {
    point := Point{}
    if flag {
        point.X = 1
    }
    return point
}

func read(point *Point) int { return point.X }

// Inside a function:
point := makePoint(true)
p := &point
x := read(p)
point.X = 2 // Allowed after the last read.
```

In Rust terminology, the constructor returns `T` and the reader accepts `&T`.

Every accepted reader follows the same contract: its references are shared and
do not escape the call. Calls must be direct, synchronous, and typed. The
checker verifies the callee too, including callees in other packages.
Calls that pass or return protected values must name the function directly.
A callback that only takes and returns scalars can receive scalar copies;
for example, `n := callback(point.X)`. It must not capture protected storage,
including an owned marked value:

```go
// Rejected: the closure captures the owner, even though it returns int.
callback := func() int { return point.X }

// Allowed: copy the scalar before creating the closure.
x := point.X
callback := func() int { return x }
```

There is no `Box<T>` ownership model. For protected types, `new(T)` and `&T{}`
are rejected. Use `value := T{}` followed by `pointer := &value`. Constructors
return `T`, never `*T`.

## What you can do with a reference

This list is **exhaustive** for references to protected storage. An owned local
can be a marked struct, a frozen collection, or an array containing protected
values. A field reached through that local can be borrowed when its type also
contains protected values. Scalar fields and scalar-only arrays cannot be
borrowed through a protected owner. Anything not listed is rejected.

| You can… | Go example |
| --- | --- |
| Initialize a fixed local reference | `var p *Point = &point` |
| Borrow a local marked value | `p := &point` |
| Borrow a local frozen collection | `p := &frozen` |
| Borrow a whole local array of protected values | `values := [2]Point{}; p := &values` |
| Borrow a protected field of an owned local | `p := &shape.Point` or `p := &owner.Items` |
| Discard a reference | `_ = p` |
| Read a field | `x := p.X` |
| Read through an explicit array dereference | `x := (*p)[0].X` |
| Call a listed frozen collection method | `n := p.Len()` |
| Copy the whole value, then change the copy | `next := *p; next.X = 1` |
| Check whether a reference is nil | `if p != nil { ... }` |
| Compare two references | `same := p == q` |
| Pass a reference to a checked reader | `x := read(p)` |

A reader names the concrete type and only reads its argument:

```go
func read(p *Point) int {
    return p.X
}
```

You can also pass `&point` directly to `read`. The reference is used only during
that call. Use named functions for your readers. User-defined methods on
protected values are unsupported, including methods promoted from embedded fields.
Only the listed `im` methods are supported: `Get`, `At`, `Len`, `All`, and `Clone`.

While a reference is still needed, you cannot change the original value:

```go
p := &point
x := read(p) // last use of p
point.X = 1 // OK here; rejected before read(p)
```

Parentheses and Go type aliases do not change these operations. Declaring a
reader parameter such as `p *Point` is also supported, as shown above.

## What you cannot do with a reference

These examples are **not exhaustive**. The allowlist above is the complete list
of what works; an operation is not allowed just because it is absent below.

| You cannot… | Rejected Go example | Use this instead |
| --- | --- | --- |
| Copy or reassign a reference | `q := p` or `p = &other` | Declare a separate reference from its owner. |
| Call a user-defined method | `value.Read()` | Use `Read(&value)`. |
| Write through a reference | `p.X = 1` | Change the owner before borrowing, or change a value copy. |
| Return a reference | `return p` | Return `*p` as a value copy. |
| Store a reference in a field or container | `Holder{P: p}` or `[]*Point{p}` | Store value copies in a marked struct or `im` collection. |
| Store a reference in a global | `saved = p` | Keep the reference local. |
| Put a reference in an interface | `var x any = p` | Pass it to a reader with a `*Point` parameter. |
| Capture a reference in a closure | `func() { read(p) }` | Call `read(p)` directly. |
| Defer or start a borrowed call in a goroutine | `defer read(p)` or `go read(p)` | Read synchronously, then use the returned value. |
| Use implicit pointer-to-array indexing | `p[0].X` | Use `(*p)[0].X`. |
| Take a scalar field's address | `&point.X` | Read `point.X` or copy it to a separate local. |
| Take a reference to a reference | `&p` | Pass `p` directly to a named reader. |
| Build an owning pointer | `new(Point)` or `&Point{}` | Build `point := Point{}`, then borrow `&point`. |
| Pass a reference to a generic helper | `readGeneric(p)` | Use a reader with a concrete `*Point` parameter. |

The checker enforces this list of supported operations; there are no extra
rules you must uphold by hand and no escape hatch for rejected operations.

## What statements you can use

This list is **complete** for statements that use protected storage or a builder.
Use the reference and builder operations listed in this guide within these forms.

| Allowed form | Go example |
| --- | --- |
| Declare one value or builder | `value := Point{X: 1}` or `values := []int{1, 2}` |
| Declare one fixed reference from its owner | `p := &value` |
| Assign one destination | `value.X = 1` or `values[0] = 2` |
| Update one destination | `value.X++` or `value.X += 2` |
| Bind both results of the listed map reader to new locals | `value, found := frozen.Get(key)` |
| Freeze into one variable | `frozen := im.FreezeSlice(values)` |
| Freeze and discard the result | `_ = im.FreezeSlice(values)` |
| Call a named reader | `read(p)` or `n := read(&value)` |
| Return one or more values | `return value` or `return value, found` |
| Forward a named function's results directly | `return makePair()` or `takePair(makePair())` |
| Use ordinary value switches | `switch value.X { case 1: value.X = 2 }` |
| Use `if` and ordinary loops | `if ready { value.X = 1 }` or `for i := range values { values[i] = i }` |
| Assign one loop target | `for value.X = range count { ... }` |
| Use checked operations inside a `select` case body | `case <-ready: values[0] = true` |

Each write still requires an owned value with no live shared reference.
A consumed builder cannot be used again. Blocks, `break`, `continue`, and other
control flow without protected operands follow ordinary Go rules. A Freeze call must be the only
right-hand-side expression of its assignment or declaration. The two-result
`Get` form creates new locals; it cannot assign into existing protected storage.
The one-destination limit applies to assignments and declarations, not to
returns. Multiple results may contain protected values, but not protected
references. A named function can pass those results directly to another named
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
value.X, p = 1, &value

// Allowed: one destination per statement.
value.X = 1
p := &value
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
| `p := &values[0]` | Copy the element with `value := values[0]`. |
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

The checker also rejects changes to package-level marked values after their
declaration initialization.

Run `immutago ./...` or `go vet -vettool=<absolute path to immutago> ./...`.
An ordinary Go build does not run the checker.

The [architecture](arch/README.md) explains the implementation. The
[source notes](../third_party/rustc/README.md) identify the rustc algorithms
used by the implementation.
