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
The pass-local [classifier](../../internal/checker/classification.go) caches
storage and callback results in separate type graphs. Storage edges follow
fields and containers; callback edges also follow interface methods and function
parameters and results. Type-parameter constraints use a separate traversal mode.
The classifier builds every reachable edge before it propagates positive results
through reverse edges. Thus cycles cannot leave a cached false result before a
reachable protected type is found. Queries do not change source-form checks or
skip packages without markers.

[Reason](../../internal/checker/proof.go) builds one obligation per concrete
type. It records the node before visiting components, so recursive types
terminate. Reverse edges connect fields and collection arguments to their
owners. Invalid components enter a queue; their failures propagate to every
owner. A recursive cycle passes only if all reachable components pass.
Each failed owner retains its first failure cause. The proof builds the
diagnostic string once, from the requested root.

Both marker modes use this proof. Stored pointers require a `Shared` struct
target; `Scoped` pointers fail the storage proof. Type parameters fail too.
[Generic checks](../../pkg/analyzer/subset_generics.go) keep user protected
generic code outside the source subset while retaining the fixed `im` APIs.

## Source forms and dependency failures

The [operation dispatcher](../../pkg/analyzer/subset_operations.go) checks each
protected expression in three steps: its producer, its consuming operation, and
its enclosing statement. Each step must match an allowed form. Unmatched forms
fail. [Call matches](../../pkg/analyzer/subset_calls.go) resolve function identity
and actual parameter types, including tuple results and variadic elements.

Per-pass caches reuse literal type matches and compiler-constant operand
classification. Each operand still passes its use and statement checks.

[Dependency facts](../../pkg/analyzer/dependencies.go) retain failure locations,
messages, relevance, and import chains. Consumers report relevant failures at
imports and export them again. Duplicate failures use one deterministic route
through a diamond import graph. These facts carry errors, not inferred function
effects; they preserve failures that a consumer-only driver run could hide.
