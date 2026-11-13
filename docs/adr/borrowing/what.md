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

References have fixed local declarations or enter named, typed, synchronous
reader functions. Only listed `im` receiver methods are accepted.
Reject reference returns, scalar interior pointers, pointer indirection and
conversion, closure captures, interface/callback transport, reference storage,
and generic borrowing helpers. Reject reference copies and rebinding, protected
multiple assignments, nested Freeze, and protected select communication operands.

Permit native containers of marked values only as checked exclusive local
builders. Freeze consumes their handles into read-only collections without
copying; Clone returns independent mutable storage. Retain direct value iteration. Marked fields remain recursively immutable.

Unmatched source forms fail. There are no suppression directives or trust annotations.
