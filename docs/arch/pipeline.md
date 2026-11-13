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
6. Check local shared loans and writes.
7. Export failures for importers.

Each stage can report errors. The builder set permits the subset checker to
recognize native storage that has a separate ownership check; it is not an
exemption from that check. The stages share compiler identities but retain
separate traversal and analysis state. There is no common intermediate form
for all three operation, freeze, and borrow checks.

## Structural checks

[Type predicates](../../internal/checker/types.go) recognize `im.Marker`,
`FrozenSlice`, and `FrozenMap` by resolved package path and named type. Aliases
are resolved. `Contains` follows fields and container element types to find
protected storage. It uses a visited-type map to stop recursive traversal.

[The structural proof](../../internal/checker/proof.go) creates a graph for
each query. A node identifies a Go type and whether the type is being checked
as a stored value or a constraint. Reverse edges point from a component to its
owners. A failed component enters a queue; failure then propagates to owners.
Recursive components start provisionally valid. A reachable invalid field
causes the containing recursive structure to fail too.

All fields and collection arguments must pass. A generic union requires all
members to pass. An intersection can pass when one bound establishes the
required property. A separate query uses the same graph machinery to check
whether an unmarked generic field can contain protected storage.

## Operation checks

[The subset checker](../../pkg/analyzer/subset.go) builds an AST parent map,
then checks source operations and inferred types. It rejects unsupported
reference storage, returns, closures, interface transport, and native
containers outside the builder rules.

[Call checks](../../pkg/analyzer/subset_calls.go) resolve direct functions and
methods, including generic syntax. Shared arguments require direct,
synchronous, concrete typed calls. The checker does not compute arbitrary
callee effects or pointer-return summaries. Instead, it checks each callee's
source under the same restrictions and propagates package failures. Generic
calls with protected type arguments are rejected outside the trusted library.

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
