# Analysis pipeline

[Architecture index](README.md)

The [command](../../cmd/immutago/main.go) passes the public analyzer to Go's
`singlechecker` driver. The driver loads and type-checks packages. The same
analyzer can run through `go vet -vettool=...`. Analysis uses the compiler's
`go/types` objects and `TypesInfo`, not names found in source text.

The [analyzer entry point](../../pkg/analyzer/analyzer.go) processes each
package in this order:

1. Skip the trusted `im` implementation and standard library.
2. Read dependency failure facts. Prepare to record local diagnostics as facts.
3. Walk source types, declarations, imports, and collection type arguments.
4. Check native builders and their freeze operations. Return the set of builders
   to the operation checker.
5. Check operations against the supported subset.
6. Export failures for importers.

Each stage can report errors. The builder set permits the subset checker to
recognize native storage that has a separate ownership check; it is not an
exemption from that check. The stages share compiler identities but retain
separate traversal and analysis state. There is no common intermediate form
for operation and freeze checks.

## Structural checks

[Type predicates](../../internal/checker/types.go) recognize `im.Immut`, `im.ImmutNoRef`,
`FrozenSlice`, and `FrozenMap` by resolved package path and named type. Aliases
are resolved. `Contains` follows fields and container element types to find
protected storage. It uses a visited-type map to stop recursive traversal.

[The structural proof](../../internal/checker/proof.go) creates a graph for
each query. A node identifies a Go type and whether the type is being checked
as a stored value or a constraint. Reverse edges point from a component to its
owners. A failed component enters a queue; failure then propagates to owners.
Recursive components start provisionally valid. A reachable invalid field
causes the containing recursive structure to fail too.

All fields and collection arguments must pass. Stored pointer targets must be
concrete `Immut` structs. `ImmutNoRef` pointers are local-only. User generic
types and functions cannot handle protected storage; the fixed `im` generic
APIs remain supported. Ordinary scalar generic code stays outside that rule.

## Operation checks

The [guide](../rules.md) is the source-language allowlist. Its allowed
forms are complete. Rejection examples show common mistakes; they do not define
the boundary.

For each operation involving protected storage, the checker must match a complete
allowed shape using syntax and resolved types. It must then check the ownership
rule for that shape. An unmatched shape is an error. A broad match such as
"this is a method call" is not enough to accept the operation.

For example, `p := &Point{X: 1}` requires a fresh composite literal with a
proved protected type and an allowed statement context. `p := &value` fails
because it would expose rebindable variable storage. A reference copy `q := p`
and a reference result `return p` use separate allowed shapes. See the
[reference rules](../rules.md#what-you-can-do-with-a-reference).

[The subset checker](../../pkg/analyzer/subset.go) uses an AST parent map to
check expression and statement context. [Call checks](../../pkg/analyzer/subset_calls.go)
resolve the function or method by compiler identity. Named readers have concrete synchronous signatures. NoRef calls cannot
retain their references. `Immut` references may be stored and returned.
`ImmutNoRef` references cannot leave local storage or direct non-retaining calls. The only protected receiver calls are the
listed methods of the exact `im` collection types.

Tests must cover each accepted shape and nearby forms that do not match it.
Adding a feature changes the docs, matcher, proof checks, and tests together.
A written allowlist alone does not establish implementation correctness;
reviewers must check the rejecting fallback and the source forms reaching it.

## Dependency facts

[Dependency checks](../../pkg/analyzer/dependencies.go) carry an
`invalidPackage` fact with relevance, original failure locations, messages,
and import chains. This is needed because drivers can hide diagnostics from
packages that the user did not select directly.

An importer collects facts, reports relevant dependency failures at imports,
and exports failures again. Duplicate failures retain one deterministic route
through a diamond-shaped import graph. These are failure records, not positive
function effect summaries. The standard driver supplies analysis of dependency
packages and fact transport.
