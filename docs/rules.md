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

Here, **protected storage** means marked values, their fields, and frozen
collections. A **builder** is a native map or slice checked for exclusive
ownership. The builder section states which native containers are checked.
Examples with `...` in a block omit ordinary code; the ellipsis is not Go syntax.


## Owned values

An owned Go local behaves like a Rust `let mut` binding:

```rust
#[derive(Clone, Copy)]
struct Point { x: i32, y: i32 }

let mut point = Point { x: 0, y: 0 };
if condition {
    point.x = 10;
} else {
    point.x = 20;
}
point = Point { x: 30, y: 40 };
```

This permits ordinary Go construction without needing Rust's `if` expressions.
Immutability does not mean “assign each field exactly once.” It means that a
shared reference cannot observe its owner being changed.

Go structs copy on assignment. Think of the Rust `Copy` example above, not a
move of a non-`Copy` Rust value:

```rust
let mut next = point;
next.x = 99; // point is unchanged
```

## Shared borrows and NLL

Taking a pointer to an existing value creates a shared borrow:

```rust
let mut point = Point { x: 0, y: 0 };
let borrowed = &point;
let x = borrowed.x; // last use of borrowed
point.x = 1;        // allowed: the borrow has ended
```

Moving the write before the last use fails:

```rust
let borrowed = &point;
point.x = 1;         // rejected: point is still borrowed
let x = borrowed.x;
```

Writing through the reference also fails:

```rust
let borrowed = &point;
borrowed.x = 1; // rejected: this is &Point, not &mut Point
```

Each local reference is initialized once from an owned local or its marked
child. Do not copy the reference or assign another pointer to it. Branches and
loops still determine its last use.

A borrow protects the **whole owner**. Borrowing a marked child also prevents
writes to its parent's other fields. Unlike Rust, Immutago does not split loans
between disjoint fields or array elements.

## Constructors and calls

Construct and return values, then borrow a local value when needed:

```rust
fn make(flag: bool) -> Point {
    let mut point = Point { x: 0, y: 0 };
    if flag { point.x = 1; }
    point
}

fn read(point: &Point) -> i32 { point.x }

let mut point = make(true);
let borrowed = &point;
let x = read(borrowed);
point.x = 2; // allowed after the last read
```

Every accepted reader follows the same contract: its references are shared and
do not escape the call. Calls must be direct, synchronous, and typed. The
checker verifies the callee too, including callees in other packages.
Calls that pass or return protected values must name the function directly.
A callback that only takes and returns scalars follows ordinary Go rules;
for example, `n := callback(point.X)` passes a scalar copy.

There is no `Box<T>` ownership model. For protected types, `new(T)` and `&T{}`
are rejected. Use `value := T{}` followed by `pointer := &value`. Constructors
return `T`, never `*T`.

## What you can do with a reference

This list is **exhaustive**: for a marked Go type `Point`, only these reference
operations are supported. Anything not listed is rejected.

| You can… | Go example |
| --- | --- |
| Initialize a fixed local reference | `var p *Point = &point` |
| Borrow a local value | `p := &point` |
| Borrow a marked child of a local value | `p := &shape.Point` |
| Discard a reference | `_ = p` |
| Read a field | `x := p.X` |
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
| Return a value | `return value` or `return frozen` |
| Use ordinary value switches | `switch value.X { case 1: value.X = 2 }` |
| Use `if` and ordinary loops | `if ready { value.X = 1 }` or `for i := range values { values[i] = i }` |
| Assign one loop target | `for value.X = range count { ... }` |
| Use checked operations inside a `select` case body | `case <-ready: values[0] = true` |

Each write still requires an owned value with no live shared reference.
A consumed builder cannot be used again. Blocks, `break`, `continue`, and other
control flow without protected operands follow ordinary Go rules. A Freeze call must be the only
right-hand-side expression of its assignment or declaration. The two-result
`Get` form creates new locals; it cannot assign into existing protected storage.

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

## Allowed value types

Every field must recursively be a scalar, an array of allowed values, another
marked struct, or an `im.FrozenSlice` / `im.FrozenMap` with allowed elements. In Rust terms,
these are plain values without interior mutability: no `Cell`, `RefCell`, or
`UnsafeCell` equivalent.

In Go, each custom aggregate must embed `im.Marker`:

```go
import "github.com/lucasavila00/immutago/pkg/im"

type Count struct {
    im.Marker
    Values [2]int
}
```

Use this wrapper instead of `type Count [2]int`. Native pointers, slices, maps,
channels, interfaces, and functions cannot be fields of a marked struct.
Unmarked structs cannot contain marked values or `im` collections.

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
Freeze does not copy or allocate. Go still allocates the original native storage;
we do not guarantee one heap allocation for a native map of arbitrary size.

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

An inline fresh allocation or literal can be frozen directly. A builder cannot
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
Unsafe operations, reflection, and unsupported aliasing therefore fail the
check. There are no escape hatches or suppression directives.

The checker also rejects changes to package-level marked values after their
declaration initialization.

Run `immutago ./...` or `go vet -vettool=<absolute path to immutago> ./...`.
An ordinary Go build does not run the checker.

The [architecture](arch/README.md) explains the implementation. The
[source notes](../third_party/rustc/README.md) identify the rustc algorithms
used by the implementation.
