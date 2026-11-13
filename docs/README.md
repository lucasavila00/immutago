# Project reference

Start with the [guide](guide/immutability.md) to use Immutago. It defines the
complete allowed Go forms. The architecture pages explain how the checker
checks those forms. Keep rules and design explanations on these pages as the
implementation changes.

## Use Immutago

| Page | Contents |
| --- | --- |
| [Ownership guide](guide/immutability.md) | Value types, shared references, allowed operations, Freeze, Clone, and CLI commands |
| [Collection costs](performance.md) | Allocations, storage sharing, clone costs, and checker memory measurements |
| [Collection API](../pkg/im/readme.md) | Generated package and method reference |
| [Analyzer API](../pkg/analyzer/readme.md) | Integration with Go analysis tools |
| [Command reference](../cmd/immutago/readme.md) | CLI entry point |

## Understand the checker

| Page | Contents |
| --- | --- |
| [Architecture](arch/README.md) | Components and design choices |
| [Analysis pipeline](arch/pipeline.md) | Type proofs, allowed source forms, and dependency failures |
| [Ownership and storage](arch/ownership.md) | Builder consumption, fixed owners, and non-lexical loan lifetimes |
| [Validation](arch/validation.md) | Trusted code, required tests, regression examples, and proof limits |
| [Rust compiler sources](../third_party/rustc/README.md) | Algorithm sources and license information |

## Change the project

| Page | Contents |
| --- | --- |
| [Development](contrib/development.md) | Repository layout, commands, snapshots, and generated docs |
| [Reviewer requirements](reviewer-focus.md) | Check the guide, complete source matches, ownership checks, and tests together |
| [Working rules](../AGENTS.md) | Instructions for agents and contributors |

Update the guide when supported forms change. Update the relevant architecture
page when the implementation or its design changes. Record test evidence in
validation and performance pages where it applies. Use Git history for past
decisions; these pages describe the current project.
