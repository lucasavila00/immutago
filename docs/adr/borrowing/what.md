# Borrowing: WHAT

## Decision

Support Rust's **owned values (`T`)**, **shared borrows (`&T`)**, and
**non-lexical lifetimes (NLL)**. Do not support **mutable borrows (`&mut T`)**.

An owned Go local behaves like a Rust `let mut` binding. It can be built through
branches, updated, and reassigned. Taking a pointer to existing storage creates
a shared borrow:

```rust
let mut value = Point { x: 0 };
let borrowed = &value;
let x = borrowed.x; // last use
value.x = 1;        // allowed
```

Writing through `borrowed`, or changing `value` before that last use, fails.

Every field is recursively a plain immutable value. Custom Go aggregates must
embed `im.Marker`; use a marked struct around an array, not a named array type.
There is no interior mutability equivalent to Rust's `Cell` or `UnsafeCell`.

## Go differences

Go values copy; they do not use Rust's move rules. Fresh pointer allocations
start owned, like a `Box<T>`. Copying that Go pointer creates a shared borrow.
Incoming pointers are shared. Go keeps its garbage collector.

Unknown relevant effects fail explicitly. There are no escape hatches.
