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
Each failed owner retains its first failure cause. The proof builds the
diagnostic string once, from the requested root.

Both marker modes use this proof. Stored pointers require a `Shared` struct
target; `LocalRef` pointers fail the storage proof. Type parameters fail too.
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

[Dependency facts](../../pkg/analyzer/dependencies.go) contain a flat table of
package records. Each record stores local failure locations and messages, plus
its direct invalid import paths. Importers merge records by package path and
share the records without changing them. A fact contains each reachable record
once. It stores no per-error import chains.

A relevant consumer builds one lexical route to each origin and reports errors
in source-location and message order. A string builder renders the report.
Silent wrappers still export their records and relevance, so a later consumer
cannot lose a failure. These facts carry errors, not inferred function effects.

Each fact must contain the reachable records: the vet driver does not re-export
imported package facts, so `AllPackageFacts` cannot supply all indirect origins.
The flat table also prevents gob from expanding shared pointer subtrees through
each diamond path. The CLI shares records in memory; vet still serializes each
fact's reachable records. See the [cost checks](../performance.md#dependency-fact-costs).
