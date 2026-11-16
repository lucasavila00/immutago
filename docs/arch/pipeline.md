# Analysis pipeline

The command and vet driver use the same `go/analysis` pass. Its input is one
package's AST and compiler type information.

## Package pass

[The pass](../../pkg/analyzer/analyzer.go) runs these stages in order:

| Stage | Work | Result used by the next stage |
| --- | --- | --- |
| Select | Inspect direct imports, used types, and dependency facts. | Skip unrelated packages; select service checks or full im checks. |
| Read failures | Report failures saved by direct dependencies. | Retain their original causes for this package's fact. |
| Check declarations | Prove im types; check service embedding and method results. | Shared type proof cache and local diagnostics. |
| Verify generation | Compare the generated file with current compiler types. | Exact file identity for generated interior record addresses. |
| Check references and builders | Match direct Borrow calls; check native builder aliases and consumption. | Builder variable identities for source checks. |
| Match source | Check complete operations on protected storage. | Diagnostics for every unmatched form. |
| Export failure | Select one local or imported cause if the package failed. | A package fact for importers. |

Generation, builder analysis, and protected source checks run only for im
packages. [Package selection](../../pkg/analyzer/package_scope.go) shares one
visited-type set across local roots; it follows used contracts, not dependency
implementations. See the complete [package boundary](../reference/rules.md#package-boundary).

## Why a Freeze call needs three checks

```go
frozen := im.FreezeSlice(values)
```

| Check | Question answered | Evidence |
| --- | --- | --- |
| Type proof | Can the elements share storage? | The compiler-resolved element type passes Frozen. |
| Builder analysis | Can another native handle change the array? | Fresh local storage, exclusive use, and no use after Freeze. |
| Source match | Is this operation a supported form? | The exact function object, argument, local binding, and complete statement match. |

[Source matching](../../pkg/analyzer/subset_operations.go) checks the expression
that creates a value, the operation that uses it, and the complete statement.
Each restricted operation has a rejecting fallback. Frozen values can
use ordinary interfaces, generics, callbacks, and captures. Target writes and
addresses still use the protected-storage checks.

| Compiler information | Why the matcher needs it |
| --- | --- |
| Function and type identity | Aliases and shadowed names must not change which API is called. |
| Declared receiver and selected method | Detect implicit addresses, dereferences, and promoted methods. |
| Instantiated signature and result tuple | Check every argument and result binding, including variadic elements. |
| Complete generic declaration types and constraints | Classify stored types without treating the record name inside `~*Record` as a stored record. Bodies use normal source checks. |
| AST parents | Check statement context and operands that Go does not evaluate. |
| Declared type parameters | Check generic bodies once; no proof of call effects for each type argument. |

Tuple bindings use one direct typed call when a component needs restricted transport.
The matcher checks each destination's write, builder, and storage rules. It expands
tuple arguments and returned calls by compiler result position. Named results use
their declared types and the same assignment checks. Ordinary tuple components do
not inherit a Local component's storage limits.

Frozen interface conversions also prove the concrete implementation and its
selected receivers. The private interface seal alone is not that proof.
[Conversion checks](../../pkg/analyzer/subset_interfaces.go) apply at the conversion;
method bodies still receive normal source checks.

## Recursive type graphs

This Frozen proof must fail for both A and B:

```go
type A struct { im.Frozen; Next *B }
type B struct { im.Frozen; Next *A; Bad map[string]int }
```

[The prover](../../internal/checker/proof.go):

1. Create one node per compiler type and requested category before visiting its components.
2. Add reverse edges from each field or element proof to its owners.
3. Expand the reachable graph. A revisit of A reuses its node.
4. Propagate the map failure to B, then A. New owners also receive cached child failures.
5. Keep one earlier failed cause per failed node. Cache its reason and category
   repair together. Copied field repairs state the required owner change for Local data.
   A source operation can supply its own repair from the resolved types and exact
   source rule. Do not infer repairs from rendered diagnostic text.

Cause links cannot form a cycle. Query order can select a different real cause.

| Graph | Cache key | Answer propagated to owners |
| --- | --- | --- |
| Type proof | Type + Frozen, Copied, or ReadResult | A component failed the requested category. |
| [Source classification](../../internal/checker/classification.go) | Type + source | A stored component or callable contract needs im source checks. |
| Ownership classification | Type + ownership | A component owns Frozen records or Local storage; Frozen handle slots stay ordinary. |
| Transport classification | Type + transport | A component needs restricted movement; Frozen values stop this walk. |
| Callback classification | Type + callback | A parameter or result needs restricted transport. |
| Local and service-reference classification | Type + query | A component contains Local values or service references. |

Classification uses compiler types, including constraint terms and result tuples.

Both graphs finish expansion before propagation. Completed answers belong to
one package pass; compiler types must remain unchanged.

## Failures across imports

```text
leaf fails ──> wrapper fact ──> app diagnostic
               original leaf location and message survive
```

[Each failure fact](../../pkg/analyzer/dependencies.go) keeps relevance, the origin
package, one location, and the exact message. Selection uses location, message,
then origin package. Vet does not forward imported facts automatically, so each
wrapper exports the original cause itself. The app diagnostic gives
`immutago <origin>` to inspect all failures at that source.
