# Borrowing: REVIEW

## Scope

The previous implementation tried to recover aliases across much of Go. The
chosen subset rejects those escape mechanisms instead. Review the smaller
language, not compatibility with the removed features.

Owned values and local shared references remain useful without pointer-return
summaries, dynamic-call effects, or recursive reference paths. Whole-owner loans
remove field-overlap decisions. Value constructors remove owning-pointer rules.

## Required checks

Verify that every accepted reference has a visible local owner or a read-only
parameter origin. Trace every accepted reference definition through the shape
matcher. Copies, rebinding, and implicit iterator references must reject. Verify
that branches and loops preserve fixed-owner NLL without copied origin maps.

Check that forbidden transport fails where introduced, including implicit Go
conversions and method receivers. Frozen collection operations must not expose writable storage. Freeze must
consume an exclusive handle; Clone must create independent native storage.

Dependency errors must reach the selected consumer in both analyzer drivers.
Diagnostics must identify the failed rule, relevant source context, and a
concrete correction. Review the per-fixture snapshots before accepting updates.

Hosted CI supplies final cross-platform, race, lint, documentation, build, and
self-analysis evidence. Passing a narrow test does not replace those checks.
