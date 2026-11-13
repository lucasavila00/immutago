# Borrowing: WHAT

## Decision

Support owned values (`T`), local shared borrows (`&T`), and non-lexical lifetimes
(NLL). Do not support mutable borrows (`&mut T`) or owned pointers (`Box<T>`).

```rust
let mut value = make(); // make returns T
let borrowed = &value;
let n = read(borrowed); // read takes &T and cannot retain it
value.n = 1;            // allowed after the last borrow use
```

Go values copy rather than move. Owned locals can be built through branches,
updated, and reassigned. Constructors return values. Borrowing any marked child
borrows the whole owner, including its other fields.

## Explicit boundaries

References stay local or pass to direct, typed, synchronous, non-escaping readers.
Reject reference returns, scalar interior pointers, pointer indirection and
conversion, closure captures, interface/callback transport, reference storage,
and generic borrowing helpers. Reject deferred or asynchronous borrowed calls.

Permit native containers of marked values only as checked exclusive local
builders. Freeze consumes their handles into read-only collections without
copying; Clone returns independent mutable storage. Retain direct value iteration. Marked fields remain recursively immutable.

The checker enforces these boundaries. Unsupported code fails; it is never
silently skipped. There are no suppression directives or trust annotations.
