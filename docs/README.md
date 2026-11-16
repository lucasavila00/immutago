# Documentation

## Guides

| Task | Guide |
| --- | --- |
| Install, generate, and check a first program | [Get started](guides/getting-started.md) |
| Build, publish, and read a complete version | [Publish immutable data](guides/immutable-data.md) |
| Start a service, call it, and stop its work | [Run a background service](guides/services.md) |

## Reference

The [language reference](reference/rules.md) defines the complete supported Go
forms and package selection rules. Use the [glossary](reference/glossary.md) for terms.

| Area | Pages |
| --- | --- |
| im types and source forms | [Type rules](reference/types.md), [common forms](reference/common-forms.md), [Frozen](reference/frozen-records.md), [Copied](reference/copied.md), [Local](reference/local.md) |
| im storage and reads | [Frozen collections](reference/frozen.md), [Cells](reference/cell.md), [generated projections](reference/projections.md), [projection primitives](reference/projection-primitives.md) |
| im interfaces and trust | [Frozen interfaces](reference/frozen-interfaces.md), [UncheckedValue](reference/unchecked-values.md) |
| supervise | [Runtime and child groups](reference/services.md), [references](reference/service-references.md), [results](reference/service-results.md) |
| API signatures and costs | [im API](../pkg/im/readme.md), [supervise API](../pkg/supervise/readme.md), [operation costs](reference/performance.md) |

## Examples

[Delivery quotes, document search, and a market maker bot](../examples/README.md) are complete Go
applications. Each has sample data, commands, HTTP requests, and tests.

## Maintainers

- [Architecture](arch/README.md): code map, type proofs, storage checks, and runtime.
- [Development](development.md): focused checks, generated docs, and diagnostic snapshots.
- [Reviewer requirements](reviewer-focus.md): compare the reference with the matcher.
- [Working rules](../AGENTS.md) and [source checks](../scripts/README.md).
- [Analyzer API](../pkg/analyzer/readme.md) and [source attribution](../third_party/rustc/README.md).
