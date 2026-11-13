# Analysis pipeline

[Architecture](README.md)

The [command](../../cmd/immutago/main.go) uses Go's `singlechecker` driver.
The same analyzer runs under `go vet -vettool`. Both provide compiler-resolved
`go/types` identities and `TypesInfo` for each package.

## Package pass

The [entry point](../../pkg/analyzer/analyzer.go) performs these stages:

1. Skip the fixed `im` package and compiler-supplied standard library.
2. Collect dependency failure facts and start recording local diagnostics.
3. Check declarations, imports, field types, and fixed `im` type arguments.
4. Run the native builder proof and obtain its set of tracked variables.
5. Match protected operations and their enclosing statements.
6. Export accumulated failures for importers.

The builder set identifies separately checked storage. It does not exempt its
operations. Type, source-form, and builder checks keep separate state.

## Type proof

[Type predicates](../../internal/checker/types.go) resolve aliases and identify
both markers and frozen collections by package path and named type.
`Contains` follows storage components with a visited-type set.

[Reason](../../internal/checker/proof.go) builds one obligation per concrete
type. It records the node before visiting components, so recursive types
terminate. Reverse edges connect fields and collection arguments to their
owners. Invalid components enter a queue; their failures propagate to every
owner. A recursive cycle passes only if all reachable components pass.

Both marker modes use this proof. Stored pointers require an `Shared` struct
target; `LocalRef` pointers fail the storage proof. Type parameters fail too.
[Generic checks](../../pkg/analyzer/subset_generics.go) keep user protected
generic code outside the source subset while retaining the fixed `im` APIs.

## Source forms and dependency failures

The [subset checker](../../pkg/analyzer/subset.go) uses AST parents to check
complete expression and statement contexts. [Call checks](../../pkg/analyzer/subset_calls.go)
resolve function identity, receivers, arguments, and results. Each protected
operation needs an allowed shape; the default path reports an error.

[Dependency facts](../../pkg/analyzer/dependencies.go) retain failure locations,
messages, relevance, and import chains. Consumers report relevant failures at
imports and export them again. Duplicate failures use one deterministic route
through a diamond import graph. These facts carry errors, not inferred function
effects; they preserve failures that a consumer-only driver run could hide.
