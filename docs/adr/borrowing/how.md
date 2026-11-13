# Borrowing: HOW

## Analysis

Adapt rustc's place-conflict, loan, region, and dataflow algorithms to Go's
compiler-resolved types and control-flow representation. Pin the source to
[`80196be`](https://github.com/rust-lang/rust/tree/80196be0e787008b6fb14d2990aba92083cd68e1/compiler/rustc_borrowck).

A place is a storage root followed by field, index, or dereference projections.
Alias provenance survives assignments, joins, calls, returns, and captures.
Value copies create independent roots; reference copies retain their targets.

Seed reference liveness at uses. Propagate region constraints backwards through
control flow and between aliases. Compute active loans with a fixed-point
worklist. At each write, expire out-of-region loans, then check overlapping live loans
before applying overwrite kills.
Unknown indices may overlap; only definite overlap can kill a loan.

Infer function effects and returned aliases, including recursive calls. Carry
summaries across package boundaries. Verify read-only calls and factory
results from their bodies; unavailable effects are unsupported, not trusted.

## Why NLL

Non-lexical lifetimes recover write permission after a borrow's last use.
Permanent freezing would reject useful code and is not this decision.

The port retains source attribution and license notices. Go-specific lowering,
collection guarantees, and unsupported operations are documented separately from
borrow-engine behavior.
