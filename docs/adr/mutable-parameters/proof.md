# Mutable parameters: PROOF

## Behavior checks

Add unit and CLI fixtures for `SetX(pointMut *Point, x int)`. Accept mutation of
an owned local before borrowing and after its last shared use. Verify the value
change against ordinary Go execution.

Reject the same call while a shared reference remains live, including aliases,
branches, and loop-carried references. Reject overlapping mutable arguments and
mutable/shared arguments that refer to the same whole owner.

## Rejected inputs

Reject attempts to return, store, capture, defer, or asynchronously use a mutable
reference. Reject conversion of a shared reference into mutable permission.
Check that local names ending in `Mut` grant nothing, and that ordinary pointer
parameters still reject writes.

Test imported function contracts and propagation of invalid dependency proofs
through both the CLI and `go vet`.

## Regression checks

Preserve conditional construction, value copies, whole-owner loans, and shared
NLL behavior. Add slice/map alias examples that remain unsupported until a
backing-storage ownership rule exists.

## Evidence

These tests are planned, not completed. Store diagnostics in sibling `.txt`
snapshots and review updates with `-update-snapshots`. Use focused local tests
for debugging and hosted CI for full cross-platform, race, lint, and build checks.
This branch contains only the ADR.
