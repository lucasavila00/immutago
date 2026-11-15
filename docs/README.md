# Documentation

Start with [a working program](getting-started.md). Use the [glossary](glossary.md)
for terms such as Frozen, local projection, and builder.

## Immutable data (`im`)

| Need | Read |
| --- | --- |
| Understand the type categories and fields | [Type rules](types.md) |
| Return an immutable record by value and store its copies | [Copied records](copied.md) |
| Declare immutable stored data | [Frozen records](frozen-records.md) |
| Combine local projections and copied results | [Local records](local.md) |
| Generate typed field projections and read local projections | [Typed projections](projections.md) |
| Publish the current Frozen atomically | [Cells](cell.md) |
| Store read-only strings, slices, and maps | [Frozen values](frozen.md) |
| Wrap data whose immutability your package promises | [Unchecked values](unchecked-values.md) |
| Look up allowed Go forms | [Common source forms](common-forms.md) |
| Check the fixed APIs used by generated projections | [Projection primitives](projection-primitives.md) |
| Look up signatures and zero values | [im API](../pkg/im/readme.md) |

## Supervised services (`supervise`)

| Need | Read |
| --- | --- |
| Start a service and understand restarts | [Supervised services](services.md) |
| Keep and call a service across restarts | [Service references](service-references.md) |
| Create and replace groups of workers | [Child services](service-children.md) |
| Supply a logger, client, or other safe concurrent dependency | [Mutable dependencies](services.md#example-mutable-dependencies) |
| Choose result types for exported service methods | [Service results](service-results.md) |
| Look up signatures and lifecycle behavior | [supervise API](../pkg/supervise/readme.md) |

## Checker, costs, and examples

- [Install the checker](getting-started.md#install) and [run it in CI](getting-started.md#check-in-ci).
- [Checker reference](rules.md): allowed source forms, trust promises, and which packages are checked.
- [Glossary](glossary.md): names used by the guides, API references, and checker.
- [Operation costs](performance.md): im collections, `im.Cell`, and `supervise.Ref.Current`.
- [Runnable projects](../examples/README.md): configuration, worker groups, and HTTP models using both packages.

## Maintainers

- [Local review](arch/README.md#local-review): reduce how much code a reviewer must trace.
- [Architecture](arch/README.md): code map, type proofs, and builder consumption.
- [Development](development.md): focused checks and diagnostic snapshots.
- [Reviewer requirements](reviewer-focus.md): contract and implementation review.
- [Working rules](../AGENTS.md): repository instructions.
- [Analyzer API](../pkg/analyzer/readme.md): embedding the public analyzer.
- [Source attribution](../third_party/rustc/README.md): adapted graph code and licenses.
