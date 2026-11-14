# Documentation

## Learn and use

- [Get started](getting-started.md): a complete example and check commands.
- [Shared](shared.md) and [Scoped](scoped.md): choose whether pointers can be saved or returned.
- [Owner](owner.md): replace the current Scoped object while readers use the previous one.
- [Frozen collections](frozen.md): builders, Freeze, reads, and Clone.
- [Unchecked immutable values](unchecked-values.md): a trusted, unproven immutable value.
- [Supervised processes](processes.md): typed start arguments and direct calls.
- [Child processes](process-children.md): start worker groups and change them while a process runs.
- [Performance](performance.md): operation costs.

## Look up a rule or API

- [Language reference](rules.md): complete rules and which packages are checked.
- [Common forms](common-forms.md): field types, construction, reads, pointers, and statements.
- [Process forms](process-forms.md): declarations, calls, PID use, and rejected examples.
- [Collection API](../pkg/im/readme.md): signatures, zero values, and iteration.
- [Process API](../pkg/proc/readme.md): supervisors, PIDs, and restarts.

## Maintainers

- [Architecture](arch/README.md): code map, type proofs, and builder consumption.
- [Development](development.md): focused checks and diagnostic snapshots.
- [Reviewer requirements](reviewer-focus.md): contract and implementation review.
- [Working rules](../AGENTS.md): repository instructions.
- [Analyzer API](../pkg/analyzer/readme.md): embedding the public analyzer.
- [Source attribution](../third_party/rustc/README.md): adapted graph code and licenses.
