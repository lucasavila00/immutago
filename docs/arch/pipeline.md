# Analysis pipeline

[Architecture](README.md)

The [command](../../cmd/immutago/main.go) uses Go's `singlechecker` driver.
The same analyzer runs under `go vet -vettool`. Both provide compiler-resolved
`go/types` identities and `TypesInfo` for each package.

## Package pass

The [entry point](../../pkg/analyzer/analyzer.go) performs these stages:

1. Skip the fixed `im` package and compiler-supplied standard library.
2. Reject cgo packages and export their failure before checking generated wrappers.
3. Collect dependency failure facts and start recording local diagnostics.
4. Check declarations, imports, field types, and fixed `im` type arguments.
5. Run the native builder proof and obtain its set of tracked variables.
6. Match protected operations and their enclosing statements.
7. Export accumulated failures for importers.

The builder set identifies separately checked storage. It does not exempt its
operations. Type, source-form, and builder checks keep separate state.

[Cgo detection](../../pkg/analyzer/cgo.go) uses the compiler's exact generated
header as a rejection trigger. It never grants trust to generated code. Cgo's
line directives retain the source package line in both drivers; the original
`import "C"` is no longer available reliably after rewriting.

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

A pass-local [Prover](../../internal/checker/prover.go) reuses one
[obligation graph](../../internal/checker/proof.go) across field, collection,
and type-argument queries. It records each node before visiting its components.
Reverse edges carry failures to owners after graph expansion. New edges to
already failed children are retained as pending seeds. Direct terminal failures
are applied before paths through earlier failed children. A recursive cycle
passes only if all reachable components pass.

Each failed owner keeps one component label and an earlier failure cause. Only
an unfailed owner can receive a cause, so cause links cannot form a cycle.
Empty-label pointer and array steps are collapsed when a cause is assigned.
The prover renders and caches each requested root message from these witnesses.
The selected failure can depend on earlier queries; rejection does not.
`Reason` remains the one-query convenience function.

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
