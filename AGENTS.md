# Working principles

Treat analyzer changes as production code. Use Go compiler type information.
The public analyzer is in `pkg/analyzer/`, immutable values are in
`pkg/im/`, type proofs are in `internal/checker/`, and the CLI is in
`cmd/immutago/`. End-to-end tests are in `tests/`.
Read [the contract](docs/rules.md) before analyzer changes.
Use the [project reference](docs/README.md) to find the current documentation.
Keep supported forms in the guide, implementation choices in the architecture
pages, and validation evidence with the relevant tests and cost documentation.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:

- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes,
simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:

- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:

- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:

- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:

```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it
work") require constant clarification.

## 5. Complete the Requested Work

**Do the full job. Do not replace it with an easier subset.**

- Do not invent urgency, deadlines, or pressure to save time, effort, or tokens. Only the user can set those tradeoffs.
- Preserve the user's intended outcome and full scope. Simplicity means the least complexity that completely solves the problem, not less of the requested problem.
- Do not silently defer difficult requirements, leave required work to the user, or call a partial implementation complete. If a requirement needs more work, do that work.
- Investigate the actual code, data, and constraints. Fix root causes and carry changes through every affected layer needed for the requested behavior.
- Define acceptance criteria from the user's request, then verify the complete result against them. Passing narrow tests does not prove a broader goal is achieved.
- If a real blocker prevents completion, state exactly what is blocked and why. Do not manufacture a blocker or request permission to avoid already authorized work.
- When corrected, update the implementation and verification scope immediately. Acknowledging the correction or rewriting the plan is not completion.

## 6. Use STE100

Always use ASD-STE100 Simplified Technical English in all writing, including
documentation, code comments, commit messages, and responses to the user.

## 7. Never Rewrite Git History

Never change, replace, or remove an existing commit. Do not amend commits,
rebase branches, move branches with `git reset`, or use a force-push. Add a new
commit for each correction. Merge remote changes when branches have diverged.

## 8. Do Not Add Unrequested Mechanisms

Do not add safety checks, runtime behavior, abstractions, or support systems
that the user did not request. Follow each stated trust assumption. Do not add
checks for data or boundaries that the user declared trusted.

## 9. Use Hosted CI for Full Validation

Use GitHub Actions for routine tests and repository-wide validation. The local
machine is a small, slow MacBook. Use local tests, targeted analyzer runs, and
profiling to debug specific failures or performance problems as needed. Avoid
repeating broad validation locally; push the branch and monitor CI.
Check changed documentation and its local links.

## 10. Do Not Post Comments Without a Request

Do not create a GitHub issue comment, pull request comment, review comment, or
discussion reply unless the user explicitly asks for that comment. Do not post
comments under the user's identity on your own initiative.

This rule does not prevent the creation or update of an issue body or pull
request body when the user requests that issue or pull request.

## 11. Prove Immutability Before Accepting It

Check every field and collection type argument. Use resolved Go type identity,
not source spelling. Do not accept unknown cases or add escape hatches. The
one exception is `proc.UncheckedDependency[T]` from `proc.Trust` in process start arguments. It is
unchecked: the checker does not look inside; the caller promises that the value
is safe for concurrent use. The user decided this
in issue #52. Do not add other escape hatches.
Support cases that can be proved. Report unsupported cases explicitly.
Unmarked structs cannot own marked values or immutable collections.
Both `im.Shared` and `im.Scoped` require immutable objects from construction.
Reject field, element, and dereference writes to protected objects. Permit
reassignment of whole variable bindings. Protected references originate from
fresh construction, never addresses of reassignable local storage. Protected
pointers target concrete marked structs only. Reject user generics involving
protected storage, protected closure captures and callback transport, and
protected operations in go or defer. Keep the fixed generic im collection APIs.
`Shared` references can be shared, stored, and returned. `Scoped` references
stay local or pass through direct typed non-escaping calls; reject storing,
returning, or capturing them. Do not add mutable initialization, loan lifetimes,
or runtime ownership state. Keep unsupported transport explicit instead of
inferring arbitrary callback effects.
Frozen collections must not expose writable storage. Freeze consumes a proved
exclusive native builder without a copy. Reject builder aliases and use after
freeze. Clone returns independent mutable storage; there are no frozen updates.
Test accepted, rejected, and unsupported programs through the real CLI.
Keep runtime checks out of the immutability proof. Record collection costs and
check performance with benchmarks.

## 12. The User Merges Pull Requests

Never merge a pull request or enable automatic merging. Create the pull request,
complete its checks and review, and mark it ready for review. The user merges it.

## 13. Keep Public APIs Pleasant to Use

Prefer concise package names and calls that read naturally in user code.
Use `im` for the immutable value package, so imports need no routine alias.
Keep examples idiomatic and convenient. Preserve the immutability guarantees
when improving the API; do not make callers repeat implementation details.

## 14. Document Exact Allowed Forms

For each supported-language rule, give a complete list of allowed Go forms
with short Go examples. State that the list is complete and that unlisted
operations are rejected. Describe rejected forms with concrete rejected and
allowed replacement examples. State that rejection examples are not exhaustive.
Do not replace these lists with abstract restrictions or explanations of intent.
Keep the documentation, checker allowlist, and acceptance/rejection tests aligned.
Distinguish rules still being implemented from behavior already verified.

## 15. Accept Documented Shapes; Reject Everything Else

The documentation defines the supported source language. For each operation on
protected storage, match the complete documented Go shape using AST nodes and
compiler type identity. Accept it only when that match and its proof checks
succeed. Report an error for every other shape. The default path must reject.
Do not accept a broad syntax category and rely on a growing list of forbidden
special cases. Check implicit receiver operations, result bindings, and the
statement context as part of the shape. A new source form requires a docs entry,
a matcher, proof rules, acceptance tests, and tests that reject nearby forms.
Keep ordinary Go code outside protected-storage analysis unaffected.

## 16. Review the Contract Against the Matcher

Follow [the reviewer requirements](docs/reviewer-focus.md) for every checker or
supported-language change. Reviewers must verify both required discussions:
complete allowed-form documentation with concrete examples, and implementation
by complete shape matches with a rejecting fallback. Trace each allowed form
through docs, code, ownership checks, CLI fixtures, and diagnostic snapshots.
A forbidden form passing the checker is a defect even when its example is not
listed in the docs. Passing known rejection tests does not replace this review.

## 17. Fix the Class of Issue

For every checker defect, identify why the operation missed a matcher or passed
an incomplete match. Fix the general classification or matching rule. Check
related forms, including declarations, nil and zero values, aliases, nested
storage, tuples, and variadic arguments where relevant. Add accepted and rejected
regressions. Do not replace a class-wide fix with an exception for one example.

## 18. Remove Unneeded Features Before Adding Machinery

When a bug or rejected program exposes a difficult Go case, first ask whether
we need that source form. A valid Go program is not automatically part of our
supported language. An existing acceptance test is evidence of current behavior,
not a requirement to preserve it at any cost.

Before extending a proof or matcher:

1. Show the failing form and an ordinary Go replacement, if one exists.
2. Compare the code and analysis needed to support each form. Include allocations
   or runtime costs when they affect the choice.
3. Prefer rejecting the replaceable form when that removes a whole class of
   alias, effect, or evaluation-order reasoning. State the changed restriction.
4. Change the complete allowed list, matcher, accepted/rejected tests, and CLI
   snapshots together. Delete analysis made unnecessary by the restriction.
5. If the required feature stays, fix its general rule and data structure. Do not
   add an exception for one spelling or discard required behavior just to pass CI.

These examples show how the current design avoids unnecessary machinery. Read
the [language reference](docs/rules.md) for the complete supported forms.

### Source forms we reject

| Example and failure | Decision and fix |
| --- | --- |
| An unmarked struct containing a marked child makes writes such as `a.b.c.d = 1` require rules about aliases into nested storage. | Require a marker on each struct that stores protected data, except the explicitly listed Owner and process forms. Prove the nested fields instead of supporting arbitrary mutable owners of immutable children. |
| `type Count [2]int` can have a mutating pointer method. Treating it as an immutable field required more method analysis. | Use `type Count struct { im.Shared; Values [2]int }`. Require a marker on each nested struct and prove its fields. The inline array does not need a separate allocation. |
| `p.Inc()` could call an embedded field's pointer method and mutate a protected value. The broad "method call" category missed the implicit receiver. | Reject user-defined methods involving protected values; use a named reader such as `Read(p)`. Keep only explicitly supported method families, including the listed process methods. |
| `case _, values[0] = <-ready` could write after another case's channel expression froze `values`. The control-flow graph did not describe that ordering. | Reject protected operations in select channel expressions and receive targets. Receive first, then change the builder in the case body. Do not add a complete select evaluation model for these forms. |
| `value, p = next, makePointer()` and `use(im.FreezeSlice(values))` mixed writes or consumption with other operations. | Use one assignment destination and a separate Freeze statement: `frozen := im.FreezeSlice(values); use(frozen)`. Keep the explicitly listed result exceptions, such as `Get`. |
| Mutable construction followed by borrowing still required NLL. Also, `p := &value; value = other` changes the storage observed by `p`. | Make marked objects read-only from construction; create pointers with `&T{...}` or supported `new` forms, not `&value`. Prepare field values in ordinary locals first. Remove the loan machinery; retain Freeze builder checks. |
| Protected `**T`, interior pointers such as `&value.N`, user generics, callbacks, captures, and go/defer calls enlarged the alias problem. | Keep concrete marked pointer targets and direct typed calls. Read pointers through those forms rather than rejecting all pointers. Copy scalar data before a callback. Support only the listed fixed library generic APIs; reject other protected forms. |
| `values := []int(nil)` was rejected although the docs loosely said "nil" was supported ([#17](https://github.com/lucasavila00/immutago/issues/17)). The first fix added a shared matcher and an 88-case syntax/type matrix. | The user rejected the feature as unnecessary. Remove the new matcher, accept `var values []int` or `im.FreezeSlice[int](nil)`, and explicitly reject typed-nil conversions. Both alternatives have the same allocation cost. [PR #18](https://github.com/lucasavila00/immutago/pull/18) fixed the documented boundary instead of extending the syntax. |
| Cgo produced many errors in generated files, and vet did not provide the original `import "C"` location ([#21](https://github.com/lucasavila00/immutago/issues/21)). | Reject a checked cgo package once at its source package line. Do not reconstruct an unavailable import location or analyze all generated wrappers. Ordinary packages outside the checking boundary follow their separate rules. |

### Requirements we removed or narrowed

| Failure or proposed complexity | Decision and fix |
| --- | --- |
| A copying `NewMap(map)` and immutable update API added collection copies. Mutable SDK parameters alone did not prove ownership of slice/map backing storage. | Use native local builders, Freeze without copying, read-only FrozenMap/FrozenSlice, and Clone for a new editable version. Remove frozen update methods. Do not build a general mutable-borrow or persistent-tree system to provide this API. |
| The proof cache replayed old depth-first/FIFO discovery to preserve the exact historical diagnostic cause. That introduced repeated walks for invalid roots. | Exact cause order was never required. Keep one real failure cause and test correct rejection and terminating cause paths. Remove replay and compatibility machinery; update snapshots if needed. [PR #29](https://github.com/lucasavila00/immutago/pull/29) records the change. |
| Each dependency fact carried every reachable failure, so vet repeatedly wrote and read the same closure of errors. | Carry one representative original failure and a command to check its origin. Do not preserve the whole error closure or add compression merely to retain it. [PR #30](https://github.com/lucasavila00/immutago/pull/30). |
| Ordinary libraries using unsafe or reflection made unrelated callers fail. A proposed fix would track where strings came from to defend against libraries changing string bytes. | Trust ordinary Go API contracts. Select checked packages from the documented imports/types/failure facts, using one compiler-type traversal. Do not add string taint analysis. [PR #46](https://github.com/lucasavila00/immutago/pull/46). |
| The process feature claimed that no mutable memory could cross a process boundary, but ordinary method bodies could use a global map. Enforcing the claim required a new language for isolated method bodies. | Check the listed process boundaries and state their limits. Do not claim full isolation or add checks beyond the agreed trust assumptions. [PR #45](https://github.com/lucasavila00/immutago/pull/45). |
| Duplicate push/PR jobs and always-on performance runs made routine CI large. | Keep four routine checks and run performance workflows manually. Keep focused local debugging; use hosted CI for full validation. [PR #26](https://github.com/lucasavila00/immutago/pull/26). |

### Simplify the implementation when the feature is needed

Do not use the examples above to reject useful supported code indiscriminately.
These fixes kept the behavior and removed repeated work:

- A scalar Clone that is never frozen follows ordinary Go container rules. Apply
  builder ownership checks only when protected elements or a later Freeze require
  them; do not restrict every use of a native slice or map.
- Type proofs and Owner queries revisited the same type graph. Reuse completed
  proofs and read the existing cache during recursive walks; test cycles and
  query order. Do not add another graph framework when the current cache suffices.
- Storage projection checks walked the same selector/index chain repeatedly.
  Walk that chain once. The duplicate classifier and duplicate Owner/source
  context were also removed instead of maintained in parallel.
- Each builder event changed one owner, but adapters built singleton gen/kill
  maps. Pass the owner ID and consume/reset flags directly to the bitset solver.
- Broad AST-category checks missed interface destinations, implicit receivers,
  closure captures, and range targets. Match the complete documented operation,
  including its types and statement context. Do not keep a permissive category
  and add one negative check for each newly found bypass.

Require a reproduction and a relevant before/after measurement for a performance
claim. Keep measurements in the PR, not in user guides. A smaller diff alone does
not prove a better algorithm.

### Current example: compiler-constant operands

During PR #60 review, a new array proof rejected this previously accepted form:

```go
// Local is a Scoped struct; p is a local *Local.
f := func() int { return len([1]*Local{p}) }
```

Go does not evaluate that array operand. Preserving the acceptance involved the
checker's unevaluated-expression exception and caches. The initial response was
to restore acceptance because the tests expected it. The user asked whether the
feature was needed at all. A replacement can compute an ordinary length before
the closure, or use the known array length.

At the time of this entry, removing that exception is a proposal, not an
implemented rule. The current reference still allows compiler-constant operands.
Before adding another exception, compare rejecting protected references in those
operands with removing the shared evaluation helper and its caches. Check type
lengths, generic constants, closure captures, and Freeze together. If the rule
changes, update the docs and tests as one change; do not quietly change snapshots
or claim the proposal is already supported.
