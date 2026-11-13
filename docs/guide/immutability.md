# The Rust subset

Immutago checks a subset of Rust's ownership rules in Go:

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
release the others. Branches, loops, deferred calls, and closures can extend a
borrow's lifetime. Disjoint fields and constant array indices can be borrowed
separately.

Incoming Go pointers are shared references. A helper can read a caller's value;
it cannot receive the equivalent of `&mut Point`.

## Constructors

A fresh Go allocation, such as `new(Point)` or `&Point{}`, starts as owned
storage. Think of an owned Rust `Box<Point>`:

```rust
fn make() -> Box<Point> {
    let mut owner = Box::new(Point { x: 0, y: 0 });
    owner.x = 1;
    owner
}
```

A checked factory can return ownership. A function returning an input reference
returns a borrow. Copying an owning Go pointer creates a shared borrow; Go does
not move the pointer as Rust moves a `Box`.

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
snapshots. `Values()` returns a native copy. See [collection costs](../performance.md).

## What we do not implement

There are no mutable borrows, lifetime annotations, partial moves, or Rust
resource ownership and destruction rules. Go still supplies its own value-copy
semantics and garbage collector.

Calls must preserve ownership and borrow information, including across package
boundaries. Unsupported alias transport, unsafe operations, reflection, and
unproved relevant effects fail explicitly. There are no user trust annotations
or suppression directives. Package-level marked values are read-only after
declaration initialization.

Run `immutago ./...` or `go vet -vettool=<absolute path to immutago> ./...`.
An ordinary Go build does not run the checker.

The [ADR](../adr/borrowing/what.md) records the decision. The
[source notes](../../third_party/rustc/README.md) identify the rustc algorithms
used by the implementation.
