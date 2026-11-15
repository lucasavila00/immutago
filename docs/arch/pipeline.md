# Analysis pipeline

The analyzer receives one type-checked Go package through `go/analysis`.
`cmd/immutago` uses the singlechecker driver; `go vet -vettool` supplies the same
analyzer through vet. Both provide AST nodes, compiler objects, resolved types,
method selections, and generic instances. Checks use those identities so aliases,
shadowed names, and Go's implicit receiver adjustments cannot change API meaning.

## Package pass

[The pass entry point](../../pkg/analyzer/analyzer.go) makes these decisions in
order:

1. Skip the fixed im and supervise implementations, standard-library source,
   and the generated test main. User test packages remain checked.
2. Read failures from direct dependency facts and determine package relevance.
   Return if neither local API use nor a relevant dependency selects the package.
3. Determine whether the package imports or uses im. If so, reject cgo before
   other checks. The compiler-generated cgo header identifies the original
   source position even when wrappers no longer contain `import "C"`.
4. Report failed direct dependencies. Capture subsequent local diagnostics for
   the fact that this package will export.
5. Walk declarations and resolved expressions. All selected packages get service
   embedding and result checks. An im package also gets recursive field proofs,
   im type-argument checks, and Go-body requirements for protected signatures.
6. Build the source checker's parent-node map. For an im package, verify the
   canonical projection file before any source operation can use its permissions.
7. Check Ref targets and direct Borrow uses in every selected package.
8. In an im package, run native builder ownership analysis, then match protected
   source operations using the returned builder set.
9. If this package or a checked dependency failed, export one representative
   original failure for importers.

A package that uses only supervise gets service checks without the full im
source and builder checks. A package with a failed dependency can report that
failure even if it has no direct import of im or supervise.

[Package selection](../../pkg/analyzer/package_scope.go) starts from direct API
imports and types used in expressions, declarations, object uses, and generic
instances. It follows aliases and named-type metadata before underlying storage,
plus used signatures, constraints, and interface contracts. One seen set is shared
across roots. It does not scan every import or all methods of an unused concrete
type. An ordinary scalar API does not select all the implementation behind it.
The im-only and runtime-inclusive walks use the same traversal with different
selection modes.

## Trace one operation

For `frozen := im.FreezeSlice(values)`, no single check is sufficient:

- Type proof checks the element type of the resulting FrozenSlice. A slice of
  mutable pointers cannot become safe merely because its handle is consumed.
- Builder discovery checks that values comes from fresh local native storage.
  Use checks reject aliases and escapes; flow checks reject use after Freeze.
- Source matching checks the exact library call, its argument, the local result
  binding, and the enclosing statement. A correct type does not permit a saved
  Freeze function or an unsupported statement context.

The type prover and source matcher share completed type proofs. Builder
analysis keeps separate state because it proves variable ownership over time,
not a property of the element type. It returns variable identities to the source
matcher without granting an exemption from normal operation checks.

## Recursive type proofs

[Prover](../../internal/checker/prover.go) answers Frozen, Copied, and ReadResult
queries. These categories describe shared payloads, independent copied values,
and restricted read results. Frozen fields, Cell, and Projection all use the
same Frozen obligation. Local fields and service data outputs use the same
ReadResult obligation. There is no separate service-field proof.

[The proof graph](../../internal/checker/proof.go) records one obligation for each
resolved type and category. It inserts a node before inspecting components.
A field, element, pointer target, or type argument adds a dependency edge, with
a reverse edge back to its owner. A direct unsupported component seeds failure.
After graph expansion, failure travels through reverse edges to every owner.
A recursive cycle succeeds only when all reachable obligations succeed.

For example, mutually recursive Frozen records A and B cannot hide a mutable
map in B. The map fails; failure reaches B and then A through the reverse
edges. Returning early at an unfinished A query would incorrectly cache a
success, so expansion and propagation are separate steps.

Completed nodes remain in the per-pass cache. New edges to already failed
children are retained as pending seeds. Direct failures are applied before
indirect paths through previously failed children. Each failed node keeps one
component label and an earlier failed cause. Only an unfailed node can receive
that cause, so the message chain terminates even when the type graph cycles.
Empty pointer and array labels are collapsed. Rendered root messages are cached.
Query order can change which real failure is shown, but not the proof result.

## Classification is different from proof

[Classifier](../../internal/checker/classification.go) asks whether a type
contains something that needs a check. It does not establish that the type is
valid. It caches separate queries for protected storage, Local values,
callbacks carrying protected storage, service references, and type-parameter
constraints. Copied records use ordinary storage and do not become protected
merely because they have a marker.

Classification also records nodes before following edges, expands all reachable
nodes, then propagates positive answers backwards. This avoids a false negative
when a cycle reaches protected storage only after an earlier recursive visit.
Different modes follow different edges: stored fields and containers matter to
storage queries; function parameters, results, and interface methods also matter
to callback and service-reference queries. Caches are owned by one pass and
require complete compiler types that do not change.

## Complete source matches

[The operation dispatcher](../../pkg/analyzer/subset_operations.go) requires
three matches for each protected expression: its producer, its consuming use,
and its complete enclosing statement. Each matcher has a rejecting fallback.
The dispatcher excludes type syntax and field-name metadata, but checks real
operands even when Go does not evaluate them, such as constant array len
operands. Capture, placement, generic, and Freeze rules still apply there.

Calls resolve the function object, instantiated parameter types, tuple results,
and variadic elements. A direct typed call can bind all results to local names
or blanks through `:=`, `=`, or `var`. The shared result matcher requires one
call and the exact compiler result count, then applies normal write and
transport checks to each component. Service signatures have their separate
result count and category limit.

[Method matching](../../pkg/analyzer/subset_methods.go) inspects the declared
function signature, not only the selector signature: Go can adjust a selector
receiver with an implicit address or dereference. Marked user receivers require
a proved non-generic type, a direct unpromoted call, and exact written receiver
identity. Receiver checks occur before the ordinary-call path, including when
the method returns only a scalar. Ordinary concrete methods can take or return
protected values through direct non-generic calls; their receivers need no marker.

[Storage matching](../../pkg/analyzer/subset_shapes.go) walks each expression's
selector, index, dereference, and parenthesis chain to find protected owners or
native builders. Thus a scalar field write still carries its owner's rule.
Cell slots have their own source context to prevent copies of atomic storage.
The source checker caches literal validation, but still checks every operand.
User generics carrying protected storage are rejected; fixed im generic APIs
use their reviewed contracts. Ordinary Copied operations keep Go rules.

### Frozen interface conversions

A Frozen interface's private im seal identifies the interface category. It does
not prove a concrete implementation. [Conversion checks](../../pkg/analyzer/subset_interfaces.go)
prove each concrete source and its selected methods at implicit or explicit
conversion. The source must be a named Frozen record or a pointer to one.
Selected user methods must be declared on that exact source; promotion and
implicit dereference fail. The promoted private seal is the only exception.
A conversion between proved Frozen interfaces preserves the earlier proof.

The interface proof requires exported methods with fixed concrete parameters.
Method results use normal source and transport checks, not stored-field proof
or a separate result count limit. Generated interface projections read and
forward these declared calls only in the verified generated file. Method bodies
still receive normal checks; there is no effect or result-origin analysis.

## Failure facts across packages

[Dependency facts](../../pkg/analyzer/dependencies.go) let a consumer fail when
a checked dependency has failed, even when the driver hides dependency
diagnostics. A fact carries relevance plus one original package, portable
source location, and exact message. It contains no method effects.

For an invalid leaf imported through a wrapper, the wrapper selects a cause
from its local and imported failures and exports that original cause again.
Selection takes the minimum location, then message, then origin package. This
makes selection independent of import order and shared graph paths. The consumer
reports one cause at each failed direct import and gives `immutago <origin>`
as the command to inspect all errors at the source.

The fact must include the cause itself. Vet does not re-export imported facts,
so a later consumer cannot depend on finding an indirect origin in the driver's
fact map. Keeping one cause also avoids copying every dependency failure into
every importer. Local diagnostics remain complete; the fact is a representative
failure, not a complete error list.

[Proof tests](../../internal/checker/prover_test.go) cover cycles and terminating
causes. [Shape tests](../../pkg/analyzer/shape_convergence_test.go) cover nearby
source forms. [Dependency graph tests](../../pkg/analyzer/dependencies_graph_test.go)
cover deterministic facts and serialization. The [source reference](../reference/rules.md)
defines the allowed language these matches implement.
