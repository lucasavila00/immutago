# The Rust subset

For marked values, their fields, and `im` collections, Immutago checks this
subset of Rust's ownership rules:

- **Owned values (`T`)** can be initialized, changed, and reassigned.
- **Shared borrows (`&T`)** permit reads, never writes.
- **Non-lexical lifetimes (NLL)** end a borrow after its last use.
- **Mutable borrows (`&mut T`)** are not supported.

The Rust below explains the rules. It is not an API you need to implement.

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

Copies of a reference share its borrow. Reassigning one reference does not
release the others. Branches and loops determine the last use.

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

There is no `Box<T>` ownership model. For protected types, `new(T)` and `&T{}`
are rejected. Use `value := T{}` followed by `pointer := &value`. Constructors
return `T`, never `*T`.

## What you can do with a reference

This list is **exhaustive**: for a marked Go type `Point`, only these reference
operations are supported. Anything not listed is rejected.

| You can… | Go example |
| --- | --- |
| Declare a local reference | `var p *Point` |
| Borrow a local value | `p := &point` |
| Borrow a marked child of a local value | `p := &shape.Point` |
| Copy a reference | `q := p` |
| Discard a reference | `_ = p` |
| Point a local reference at another local value | `p = &other` |
| Clear a local reference | `p = nil` |
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
that call. Methods with a `*Point` receiver follow the same rule.

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
| Write through a reference | `p.X = 1` | Change the owner before borrowing, or change a value copy. |
| Return a reference | `return p` | Return `*p` as a value copy. |
| Store a reference in a field or container | `Holder{P: p}` or `[]*Point{p}` | Store value copies in a marked struct or `im` collection. |
| Store a reference in a global | `saved = p` | Keep the reference local. |
| Put a reference in an interface | `var x any = p` | Pass it to a reader with a `*Point` parameter. |
| Capture a reference in a closure | `func() { read(p) }` | Call `read(p)` directly. |
| Defer or start a borrowed call in a goroutine | `defer read(p)` or `go read(p)` | Read synchronously, then use the returned value. |
| Take a scalar field's address | `&point.X` | Read `point.X` or copy it to a separate local. |
| Take a reference to a reference | `&p` | Copy the reference with `q := p`. |
| Build an owning pointer | `new(Point)` or `&Point{}` | Build `point := Point{}`, then borrow `&point`. |
| Pass a reference to a generic helper | `readGeneric(p)` | Use a reader with a concrete `*Point` parameter. |

The checker enforces this list of supported operations; there are no extra
rules you must uphold by hand and no escape hatch for rejected operations.

## Allowed value types

Every field must recursively be a scalar, an array of allowed values, another
marked struct, or an `im.Slice` / `im.Map` with allowed elements. In Rust terms,
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

The `im` collections share immutable backing storage. Updates return new
snapshots. Native slices, maps, and channels containing marked values are
rejected. `Values()` is therefore unavailable for marked elements.

Pass values directly to `im.NewSlice`. `im.NewMap` and `im.NewSlice` also accept
an inline native container literal as their immediate input: the library copies
it before publishing storage. Direct `range` over `All()` yields value copies;
do not store or pass the iterator function. Scalar collection APIs remain
available. See [collection costs](../performance.md).

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

The [ADR](../adr/borrowing/what.md) records the decision. The
[source notes](../../third_party/rustc/README.md) identify the rustc algorithms
used by the implementation.
