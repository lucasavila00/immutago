# Documentation

## Immutable data (`im`)

| Need | Read |
| --- | --- |
| Install and run a working program | [Get started](getting-started.md) |
| Store or return pointers to read-only objects | [Shared](shared.md) |
| Prevent functions from keeping object pointers | [Scoped](scoped.md) |
| Replace a pointer while readers use the previous object | [Owner](owner.md) |
| Build read-only slices and maps | [Frozen collections](frozen.md) |
| Wrap data whose immutability your package promises | [Unchecked immutable values](unchecked-values.md) |
| Choose an immutable return type | [Scoped results](scoped-results.md) |
| Look up allowed Go forms | [Common source forms](common-forms.md) |
| Look up signatures and zero values | [im API](../pkg/im/readme.md) |

## Supervised services (`supervise`)

| Need | Read |
| --- | --- |
| Start a service and understand restarts | [Supervised services](services.md) |
| Create and replace groups of workers | [Child services](service-children.md) |
| Supply a logger, client, or other safe concurrent dependency | [Mutable dependencies](services.md#mutable-dependencies) |
| Check method result types | [Scoped results](scoped-results.md) |
| Look up signatures and lifecycle behavior | [supervise API](../pkg/supervise/readme.md) |

## Checker, costs, and examples

- [Install the checker](getting-started.md#install) and [run it in CI](getting-started.md#check-in-ci).
- [Checker reference](rules.md): allowed source forms, trust promises, and which packages are checked.
- [Operation costs](performance.md): im collections, `im.Owner`, and `supervise.Ref.Current`.
- [Runnable projects](../examples/README.md): configuration, worker groups, and HTTP models using both packages.

## Maintainers

- [Local review](arch/README.md#local-review): reduce how much code a reviewer must trace.
- [Architecture](arch/README.md): code map, type proofs, and builder consumption.
- [Development](development.md): focused checks and diagnostic snapshots.
- [Reviewer requirements](reviewer-focus.md): contract and implementation review.
- [Working rules](../AGENTS.md): repository instructions.
- [Analyzer API](../pkg/analyzer/readme.md): embedding the public analyzer.
- [Source attribution](../third_party/rustc/README.md): adapted graph code and licenses.
