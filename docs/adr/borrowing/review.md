# Borrowing: REVIEW

## Contract

Shared parameters are read-only. Factories construct their own storage; value
copies own independent storage. Owner writes become valid after the last shared
use. Pointer rebinding does not overwrite the old target.

## Source review

Compared the port with the pinned rustc liveness, region, conflict, and ranked
worklist code. Region expiration precedes access checks; overwrite kills follow
them. Possible overlap checks conflicts; definite overlap permits kills. Alias
copies share one region. The general SCC solver is tested independently.

The review found and corrected shared analysis-state mutation, lost returned
global aliases, loop-site lifetime truncation, and unsafe widening across
pointer dereferences. Unknown effects now survive helper summaries. Unsupported
aggregate, channel, closure, panic, and dynamic-call paths cannot silently drop
protected references. Returned global identities conservatively overlap during
initialization when exact identity is unavailable.

## Evidence

Targeted debugging accepted owned construction, independent copies, fresh
factories, and writes after the final shared use. Rejection regressions cover
scalar aliases, type erasure, transport wrappers, deferred closures, loops,
recursive summaries, and initializer aliases. Randomized reference solvers
check dataflow and region propagation. The initializer regression passes its
focused test. Full cross-platform, race, lint, documentation, and repository
checks run in hosted CI.
