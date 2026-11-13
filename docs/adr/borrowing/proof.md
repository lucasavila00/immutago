# Borrowing: PROOF

## Test the Rust rules

Accept owned construction, reassignment, independent copies, fresh constructors,
and reads through shared references. Accept a write after the last shared use:

```rust
let borrowed = &value;
read(borrowed);
value.x = 1; // allowed after the call
```

Reject the same write while the borrow is still live:

```rust
let borrowed = &value;
value.x = 1; // rejected
read(borrowed);
```

Reject writes through shared references. There is no `&mut T` operation that
could authorize them.

## Cover Go's alias paths

Test branches, loops, disjoint fields, array indices, pointer rebinding, generic
helpers, cross-package calls, returned references, closures, deferred calls, and
goroutines. Test scalar pointers into marked structs, not only `*Marked` types.
Unsupported paths must fail instead of losing a borrow.

Compare the ported graph algorithms with independent reference solvers. Check
that recursive paths terminate. Keep collection snapshot and allocation tests.

## Validation

Use local runs to debug specific failures. Run the full race-enabled suite on
Linux, macOS, and Windows in CI, together with lint, generated documentation,
CLI build, and repository self-analysis.
