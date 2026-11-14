# Analysis pipeline

The [command](../../cmd/immutago/main.go) uses Go's `singlechecker` driver.
The same analyzer runs under `go vet -vettool`. Both provide compiler-resolved
`go/types` identities and `TypesInfo` for each package.

## Package pass

The [entry point](../../pkg/analyzer/analyzer.go) performs these stages:

1. Skip the fixed `im` and `proc` packages and compiler-supplied standard library.
2. Collect dependency facts and determine the checked package boundary.
3. Return for ordinary packages under normal Go API trust.
4. Reject checked cgo packages and export their failure before checking wrappers.
5. Report checked dependency failures and record local diagnostics.
6. Check declarations, imports, field types, exported process methods, package
   variables in process methods, and fixed `im` and `proc` type arguments.
   Export one fact for each function that uses a rejected package variable.
7. Run the native builder proof and match protected source forms.
8. Export accumulated failures for importers.

The builder set identifies separately checked storage. It does not exempt its
operations. Type, source-form, and builder checks keep separate state.

[Package scope](../../pkg/analyzer/package_scope.go) uses direct `im` or `proc` imports,
locally used compiler type contracts, and failed checked dependency facts. One
existential type walk shares a seen set across expression and object roots.
It inspects alias and named generic metadata before underlying types, selected
signature receivers and constraints, and explicit interface methods and embedded
terms. It does not walk all imports or unselected concrete method sets. This
keeps scalar APIs ordinary even when their implementation uses marked types,
and preserves the boundary when vet prunes imported type data.

Ordinary code is trusted to respect Go API contracts, including immutable
strings and no hidden writes to protected memory. This is a conditional proof
boundary, not a string dataflow or memory-corruption proof. The
[language contract](../rules.md#package-boundary) owns the exact selection rule.

[Cgo detection](../../pkg/analyzer/cgo.go) uses the compiler's exact generated
header as a rejection trigger within checked packages. It never grants trust to generated code. Cgo's
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
classification. Each operand still passes its use and statement checks. Storage
classification follows one selector, index, dereference, or parenthesis chain,
checking resolved types and tracked builder identities at each step.

[Dependency facts](../../pkg/analyzer/dependencies.go) contain one original
failure: its package path, portable source location, and exact message. Each
package selects the first cause by source location, message, and origin package.
The minimum of the imported witnesses and local errors is also the first cause
in the full failing dependency graph. Shared paths and import order do not change
the choice. The relevance bit is collected separately from this choice.

A failed fact still crosses silent wrappers. Relevant consumers report one
representative cause for each failed direct import. The report names the
original package to check, because running a silent wrapper can print no errors.
Reports from direct package analysis do not change. No original message is shortened,
but downstream reports do not enumerate all causes or show full import routes.
These facts carry errors, not inferred function effects.

[Variable use facts](../../pkg/analyzer/variable_uses.go) are object facts on
functions and concrete methods of checked packages. One walk over each function
body records rejected package variables, imported callee facts, and callees in
the same package. A worklist then marks the callers of each marked function, so
each function enters it once and recursion ends. A fact holds the first variable
found, not the call path. Process methods read these facts for direct calls and
function names. Facts exist only for marked functions; vet carries them to direct
importers, and the fact of a callee already includes the facts of its own callees.

The vet driver does not re-export imported package facts. `AllPackageFacts`
cannot supply all indirect origins, so each exported witness is self-contained.
Carrying the full error graph in each fact would repeat its payload at every
importer. One witness keeps each fact independent of the number of upstream
packages and errors. See the [cost checks](performance.md#dependency-fact-costs).
