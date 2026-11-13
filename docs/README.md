# Documentation

Immutago checks a small ownership system for Go. Marked structs contain only
immutable field types. Local owners can change; shared references can only read.
Freeze transfers a native map or slice to a read-only collection without copying.

## For users

1. [Get started](getting-started.md): run a complete example and add the checker
   to your project.
2. [Language rules](rules.md): find the complete allowed operations and rejected
   examples. This is the source of truth for supported Go forms.
3. [Costs](performance.md): understand storage sharing, allocations, and Clone.
4. [Collection API](../pkg/im/readme.md): look up signatures, zero values,
   iteration order, and bounds behavior.

## For maintainers and reviewers

| Task | Read |
| --- | --- |
| Find the code that owns a check | [Architecture](arch/README.md) |
| Follow types, calls, and dependency failures | [Analysis pipeline](arch/pipeline.md) |
| Understand loans and consumed builders | [Ownership analysis](arch/ownership.md) |
| Check assumptions and test evidence | [Validation](arch/validation.md) |
| Build, debug, or update snapshots | [Development](development.md) |
| Review a change to the supported language | [Reviewer requirements](reviewer-focus.md) |
| Work on the repository as an agent | [Working rules](../AGENTS.md) |

For integration, use the generated [analyzer API](../pkg/analyzer/readme.md)
and [CLI instructions](getting-started.md#add-checks-to-an-existing-project). The
[Rust compiler source notes](../third_party/rustc/README.md) identify the adapted
algorithms and their licenses.

## Where documentation belongs

The getting-started page teaches one complete path. The rules page defines
supported behavior. Architecture pages explain implementation choices and
invariants. Development and review pages explain how to change the project.
Keep each fact in its primary page and link to it from other pages.

Generated package references come from Go comments. Update the comments and run
`make docs`; do not edit generated Markdown. Git history preserves earlier
designs. These pages describe the current implementation.
