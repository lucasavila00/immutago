# Frozen collections: PROOF

Test accepted native construction, reads, edits, Freeze, frozen reads, Clone,
independent edits, and refreezing. Cover primitive and recursively marked values,
empty and nil storage, branches, loops, and immediate fresh Freeze inputs.

Reject aliases created before freezing; writes and reads after freezing;
subslice and element-pointer aliases; unknown calls; parameter/global sources;
closure and interface transport; double Freeze; and Freeze during active range.
Reject indirect, deferred, and concurrent Freeze calls. Invalid element types
must fail the recursive structural proof.

Exercise imported helpers and dependency-error propagation through both the CLI
and vet. Keep sibling `.txt` diagnostic snapshots with source context and fixes.

Runtime tests check clone independence and original-storage reuse. Allocation
tests separate construction from Freeze and assert zero allocations for Freeze,
reads, and direct iteration. Benchmarks report native-size-dependent clone costs.

Use focused local debugging, then hosted CI for full race tests on Linux,
macOS, and Windows, source checks, generated docs, build, and self-analysis.
