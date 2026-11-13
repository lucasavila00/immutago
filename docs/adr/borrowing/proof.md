# Borrowing: PROOF

## Required behavior

Add unit tests for place overlap, loan issuance and kills, region propagation,
and fixed-point convergence. Compare ported operations with cases from the pinned
rustc source. Include both field-disjointness and unknown-index overlap.

End-to-end tests must accept conditional construction, reassignment without a
live borrow, read-only pointer parameters and methods, independent value copies,
fresh allocation construction, and writes after the final reference use.

Reject owner writes while a borrower remains live, writes through shared
references, aliases hidden by generic helpers, unproved call effects, and writes
to published collection storage. Check branches, loop back edges, pointer
rebinding, interior aliases, closures, deferred calls, goroutines, and
cross-package returned aliases. Each rejection must identify the source operation.

## Validation

Run focused tests locally and the full race-enabled suite in hosted CI on
Linux, macOS, and Windows. Run lint, generated-documentation checks, CLI build,
and repository self-analysis. Preserve persistent collection snapshot and
allocation tests.

The existing blanket pointer rejection and its passing tests do not establish
this decision. Completion requires replacing those expectations and recording
successful checks for the implemented loan analysis.
