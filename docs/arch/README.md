# Architecture

Immutago checks a restricted set of Go operations on immutable data. It uses
compiler types to prove stored data, source matches to control access, and
control-flow analysis to consume native builders. The im runtime stores native
Go values. It adds no ownership flags or lifetime checks.

The design lets a reviewer check a data declaration and its construction path
without tracing every reader. It does not prove all effects of an application.
Ordinary Go APIs must keep their contracts. Service locks, state changes, and
input use still need review in service code.

## How data moves

A typical update has four stages:

```text
native builder -> Frozen value -> Cell -> local projection
   mutate          share         replace     read
```

A native slice or map starts as fresh local storage. Freeze transfers its
storage to an immutable wrapper without copying entries. The analyzer rejects
aliases and later use of the builder. Recursive type proofs establish that its
elements are safe to share. A Cell can atomically replace the published root.
Readers keep a projection of the root they loaded, even after replacement.
Generated field methods read that root without copying parent records.

Three record markers separate the data contracts:

- Frozen records can retain shared immutable data. Their fields cannot change
  after construction.
- Local records hold read results, including projections. Their fields also
  cannot change after construction, but the values cannot enter persistent
  storage. Local pointers have stricter transport rules.
- Copied records contain only booleans, numbers, unnamed arrays, and other
  Copied records. An independent copy can change under ordinary Go rules.

The marker is a declaration to check, not proof by itself. A numeric field in a
Frozen record still belongs to that immutable owner. A copy of the numeric value
in an ordinary variable can change. [Storage and ownership](ownership.md)
explains how the implementation separates payload types, storage owners, and
read access.

## How analysis reaches a decision

The command and vet driver give the analyzer a type-checked package. The pass
selects packages that import or use the fixed APIs, or depend on checked
failures. It proves marked types, verifies generated code, checks service
references, and checks operations on protected storage. Unsupported protected
operations fail; ordinary Go outside those operations keeps Go rules.

For example, proving `FrozenSlice[*Record]` establishes the element type. It
does not establish that the native slice passed to Freeze has no alias. Builder
analysis establishes that fact. It also does not permit every Go expression
that mentions the result: source and statement matches must pass separately.
These checks are independent because a valid type does not make an arbitrary
use safe. [The pipeline](pipeline.md) shows their order, caches, and dependency
facts.

## Services use a separate lifetime runtime

A service is a mutable struct with an embedded `supervise.Service[A]`. Start
initializes it and returns a Ref with a shared atomic slot. Borrow selects the
latest initialized value; a method call runs in the caller's goroutine. Groups
control start and stop order. One watcher retries one failed service.

The analyzer checks service targets, direct Borrow call forms, and exported
method result types. It does not prove that service state stays private or
that concurrent calls are safe. [Services](services.md) explains the checked
boundary and the runtime from initialization to shutdown.

## Source map

| Part | Entry point and responsibility |
| --- | --- |
| Drivers | [cmd/immutago](../../cmd/immutago/main.go) runs the Go analysis driver. [cmd/immutago-gen](../../cmd/immutago-gen/main.go) generates one package's typed readers. |
| Package analysis | [pkg/analyzer](../../pkg/analyzer/analyzer.go) selects checks, matches source, reports diagnostics, and exports dependency failures. |
| Type proofs | [internal/checker](../../internal/checker/prover.go) proves recursive type categories and classifies protected storage. |
| Builder analysis | [internal/freeze](../../internal/freeze/freeze.go) checks exclusive native handles and use after Freeze. |
| Generation | [internal/projectiongen](../../internal/projectiongen/generate.go) creates canonical readers from compiler types. |
| Data runtime | [pkg/im](../../pkg/im/marker.go) supplies markers, native wrappers, projections, and atomic Cells. |
| Service runtime | [pkg/supervise](../../pkg/supervise/supervise.go) supplies service arguments, references, retries, and groups. |

[Validation](validation.md) explains the trust assumptions and test layers.
[Performance](performance.md) explains runtime costs and how to measure analyzer
changes. The [language reference](../reference/rules.md) gives the complete
allowed Go forms. Architecture pages explain implementation choices; they do
not extend that source language.
