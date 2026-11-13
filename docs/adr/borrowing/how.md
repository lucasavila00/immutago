# Borrowing: HOW

## Local NLL

Use Go's typed syntax and statement control-flow graph. An owner is a local
value. A reference binding records which whole owners it can borrow.

```rust
let borrowed = &value;
let copy = borrowed;
read(copy); // both bindings refer to the same owner
value.n = 1;
```

Compute possible owners forward through assignments, branches, and loops. Trace
reference uses backward to their binding definitions with the ported rustc
liveness algorithm. A write fails if any live reference can borrow its owner.
Writes through references always fail.

Separate right-hand-side evaluation from assignment writes. Copies and pointer
rebinding must preserve Go's evaluation order, including simultaneous assignment.

## No hidden aliases

Reject escape boundaries in source. Every accepted reader has the same shared,
non-escaping contract. There is no inference of arbitrary callee effects,
returned-pointer ownership, recursive pointer paths, or dynamic callbacks.

Check dependency bodies and propagate proof failures through package facts,
including when `go vet` suppresses dependency diagnostics. Preserve the original
failure context for the importing caller.

Retain the rustc source attribution and license for the algorithms still used.
