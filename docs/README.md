# Documentation

Immutago checks immutable Go objects and exclusive native builders.
`im.Immut` permits shared references. `im.ImmutNoRef` keeps references local
or in direct calls that do not retain them. Both prohibit object mutation.

## Users

- [Get started](getting-started.md): a complete example and check commands.
- [Language rules](rules.md): the complete allowed forms and rejected examples.
- [Collection API](../pkg/im/readme.md): signatures, zero values, and iteration.
- [Performance](performance.md): storage sharing, Clone, and cost checks.

## Maintainers

- [Architecture](arch/README.md): code map, type proofs, and builder consumption.
- [Development](development.md): focused checks and diagnostic snapshots.
- [Reviewer requirements](reviewer-focus.md): contract and implementation review.
- [Working rules](../AGENTS.md): repository instructions.
- [Analyzer API](../pkg/analyzer/readme.md): embedding the public analyzer.
- [Source attribution](../third_party/rustc/README.md): adapted graph code and licenses.

The rules page defines behavior. Architecture explains its implementation.
Tests and performance notes record validation. Generated package references
come from Go comments; update those comments and run `make docs`.
