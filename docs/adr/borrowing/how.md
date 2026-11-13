# Borrowing: HOW

## Use rustc's borrow-checking algorithms

Port the relevant algorithms from the
[pinned rustc source](https://github.com/rust-lang/rust/tree/80196be0e787008b6fb14d2990aba92083cd68e1/compiler/rustc_borrowck).
Keep their source attribution and license.

A **place** identifies storage, such as `value.x`. A **loan** records a shared
borrow of that place. A **region** contains the program points where the borrow
must remain valid.

```rust
let borrowed = &value.x; // issue a loan
let x = *borrowed;       // last use in its region
value.x = 1;            // loan has ended
```

Follow uses backward through branches and loops to compute NLL. Reference copies
share the same loan. Before a write, expire loans whose regions have ended, then
reject overlap with a live loan. Only a definite overwrite can kill a loan.

## Adapt to Go

Resolve Go pointer aliases to their original storage. Preserve that identity
through assignments, calls, and returns. Export checked function effects across
packages. Distinguish a returned owner from a returned borrow.

Bound recursive reference paths conservatively. If the checker loses information
needed for a proof, report an unsupported operation. Do not grant write access.
