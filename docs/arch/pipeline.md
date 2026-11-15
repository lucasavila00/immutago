# Analysis pipeline

The [CLI](../../cmd/immutago/main.go) uses Go's `singlechecker` driver.
`go vet -vettool` runs the same analyzer. Both supply resolved `go/types`
identities and `TypesInfo` for each package.

## Package pass

The [entry point](../../pkg/analyzer/analyzer.go) runs these stages:

1. Skip the fixed im and supervise implementations and the standard library.
2. Collect dependency facts and select checked packages.
3. Return for ordinary packages.
4. Reject cgo in packages that import or use im; report one source error.
5. Report checked dependency failures and record local diagnostics.
6. Check [service method results](services.md), excluding Init, Run, and methods
   supplied by the runtime. In im packages, also check field types and im type arguments.
7. For packages that import or use im, require the exact generation directive
   and current output for source Frozen records.
8. Match service reference operations in all selected packages.
9. For packages that import or use im, prove native builder ownership and match
   protected source forms.
10. Export failures for importers.

Package selection and state:

- Type, source-form, and builder checks keep separate state. Tracking a builder
  does not exempt its operations from matching.
- [Package scope](../../pkg/analyzer/package_scope.go) implements the
  [package selection rules](../rules.md#package-boundary).
- Each selection walk shares a seen set across expression and object roots. The
  runtime-inclusive walk selects service checks; the im-only walk selects source
  and builder checks. Both use the same type traversal. It visits
  aliases and generic metadata before underlying types, selected receivers,
  constraints, interface methods, and embedded terms.
- It does not walk all imports or unselected concrete methods. Scalar APIs stay
  ordinary even when their implementation uses marked types. Vet's pruned type
  data does not change package selection.
- Ordinary code is trusted to obey Go contracts. The checker does not track
  string origins or defend against arbitrary memory writes.
- [Cgo detection](../../pkg/analyzer/cgo.go) rejects the compiler's exact generated
  header in packages that import or use im. It reports the original package line, retained by
  line directives. Generated wrappers do not reliably retain `import "C"`.

## Type proof

[Type predicates](../../internal/checker/types.go) resolve aliases and identify
`im.Frozen`, `im.Local`, and frozen collections by package path and named type.

The per-pass [classifier](../../internal/checker/classification.go) answers
whether a type contains protected storage, callbacks, or service references:

| Query | Edges followed |
| --- | --- |
| Storage | Fields and containers. |
| Local | Local records, projection wrappers, and containing storage. |
| Callback | Storage edges, interface methods, function parameters and results. |
| Service reference | Fields, containers, interface methods, function parameters and results. |
| Type-parameter constraint | Separate traversal mode. |

It expands all reachable edges before propagating positive results backwards.
This prevents a cycle from caching false before reaching a protected type.
Cached queries do not bypass source checks or package selection.

The per-pass [Prover](../../internal/checker/prover.go) checks field, collection,
and type arguments in one [graph](../../internal/checker/proof.go):

1. Record a (type, proof category) node before visiting its components.
2. Expand the graph; retain edges to already failed children as pending seeds.
3. Apply direct failures before paths through previously failed children.
4. Propagate failures through reverse edges. A cycle passes only if every
   reachable component passes.

Failure messages:

- Each failed node keeps one component label and an earlier cause.
- Only an unfailed node can receive a cause, so cause links cannot cycle.
- Collapse empty pointer/array labels; render and cache each requested message.
- Earlier queries can affect the selected cause, but not acceptance or rejection.
- `Reason` is the single-query convenience function.

Frozen, Copied, and ReadResult have separate cached obligations. Cell and
Projection use the same Frozen proof. Local fields and service results use
ReadResult. Stored pointers require a Frozen target. Type parameters fail; [generic checks](../../pkg/analyzer/subset_generics.go)
retain the fixed im APIs and reject user generics with protected storage.

## Source forms and dependency failures

The [operation dispatcher](../../pkg/analyzer/subset_operations.go) requires
three allowed-form matches for each protected expression:

1. How the value is produced.
2. How it is used.
3. The enclosing statement.

Unmatched forms fail. [Calls](../../pkg/analyzer/subset_calls.go) resolve function
identity and actual parameter types, including tuples and variadic elements.

[Marked method](../../pkg/analyzer/subset_methods.go) matching requires:

- A direct call and one selection index: no promoted method path.
- A proved, non-generic marked receiver.
- Exact identity between written and declared receiver types. Read the function
  object's signature; a selector signature can include Go adjustments.
- Receiver checks before the ordinary-call fast path. Even a scalar method can
  implicitly take a protected field's address.

Ordinary concrete receivers can also take protected inputs or return protected
values through direct, non-generic methods. Their receiver does not need a marker;
the protected values keep the same source checks.

Method bodies still pass write, pointer-use, and statement checks. Literal type
matches are cached, but every operand is checked, even when Go does not evaluate
it. Storage classification walks each selector/index/dereference/parenthesis
chain once, checking types and builder identities along the way.

[Dependency facts](../../pkg/analyzer/dependencies.go) carry errors, not function effects:

- One original failure: package path, portable location, and exact message.
- Select the minimum by location, message, then origin package across local and
  imported errors. Import order and shared paths do not change the choice.
- Record package relevance separately.
- Forward failures through wrappers, even when those wrappers print no local error.
- Report one cause per failed direct import and name the original package to check.
  Keep original messages intact; omit other causes and full import routes.

Each fact must be self-contained: vet does not re-export imported facts, so
`AllPackageFacts` cannot supply every indirect origin. One cause avoids copying
all failures at each importer. See [dependency costs](performance.md#dependency-fact-costs).
