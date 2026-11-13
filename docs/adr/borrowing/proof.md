# Borrowing: PROOF

## Accepted programs

Test conditional construction, reassignment, value-returning constructors,
independent value copies, direct readers, local reference copies and rebinding,
branches, and loops.

```rust
let borrowed = &value;
read(borrowed);
value.n = 1; // allowed after the last use
```

## Rejected programs

Reject writes through references and writes while the whole owner is borrowed:

```rust
let borrowed = &value;
value.n = 1; // rejected
read(borrowed);
```

Test every escape boundary separately: reference returns/storage, scalar
addresses, indirection, conversions, interfaces, closures, callbacks, generics,
deferred/asynchronous calls, and native containers. Test right-hand-side borrow
creation during simultaneous assignment.

## Evidence

Store exact compiler and analyzer output in sibling `.txt` fixture snapshots,
using tgo's `-update-snapshots` workflow. Compare source positions, explanations,
and suggested fixes. Exercise both the CLI and `go vet`, selecting only consumers
to verify dependency-failure propagation.

Use local runs for specific debugging. Run full race-enabled tests on Linux,
macOS, and Windows in CI, plus lint, generated docs, build, and self-analysis.
Keep collection snapshot and allocation tests.
