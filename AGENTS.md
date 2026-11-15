# Working principles

Reduce the code that a reviewer must trace. Use im type rules to check immutable
data. Review service state changes, locks, and input use in the service code.
Ordinary Go contracts still apply to callers and dependencies. Do not claim full
service isolation or add whole-program effect analysis to obtain it.

Treat analyzer changes as production code. Use Go compiler type information.
The public analyzer is in `pkg/analyzer/`, immutable values are in
`pkg/im/`, type proofs are in `internal/checker/`, and the CLI is in
`cmd/immutago/`. End-to-end tests are in `tests/`.
Read [the contract](docs/rules.md) before analyzer changes.
Use the [project reference](docs/README.md) to find the current documentation.
Use the [glossary](docs/glossary.md) for names in guides, comments, and diagnostics.
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

## 11. Prove Types Before Accepting Operations

Use compiler-resolved types and the recursive categories in [type rules](docs/types.md).
Frozen fields, Cell[T], and Projection[T] use one Frozen proof. Local fields and
service outputs use one ReadResult proof. Copy and local Clone use Copied.
Do not add a service-only field proof or a context-dependent marker meaning.
Copied permits booleans, numbers, Copied records, unnamed arrays of Copied values,
and opaque service references. Copied records contain only bool/numeric fields,
unnamed arrays, and other Copied records; no strings or reference handles. Their
fields are immutable from construction. Ordinary holders can store and replace
complete Copied values without becoming immutable. Copied pointers are rejected.
Frozen records are not Copied,
even when their fields contain only numbers. Struct results require Copied, Local,
or a generated projection.
Embed Frozen, Local, and Copied markers to declare record categories. The markers
themselves do not satisfy Frozen, Copied, or ReadResult payload proofs.

Frozen and Local fields are immutable from construction. Local values cannot
enter persistent storage, including through nested records or arrays. Local
pointers stay local or pass to direct typed calls; never store or return them.
Ordinary protected pointers originate from fresh storage. Fixed projection APIs
and verified generated getters can select record addresses inside frozen storage.
Do not add mutable construction, loan tracking, string provenance, or runtime
ownership state.

Generated getters read fields directly. Permit im.Projection.Read and interior
Frozen-record addresses only in the exact current generated file. Verify its
normalized bytes and physical token file before source checks. Handwritten Read
calls are rejected, including scalar reads. All normal type, write, capture,
generic, receiver, and transport checks still apply to generated nodes.
Generated record and array projections keep the root pointer and indices;
do not copy parent values during field reads.
CopyRecord is the explicit root copy for a record value, including a map result.
Use ProjectString for native strings and Projection methods for frozen strings,
slices, and maps. Keep generic projection adapters in the low-level reference.
Normal projection reads return Copied values or further projections. StringProjection.Unsafe and
FrozenString.Unsafe explicitly return a shared native string; trust the caller
and add no string provenance or lifetime analysis. Native collection builders keep
exclusive-use and consume-on-Freeze checks. Freeze copies no backing storage;
Clone makes explicit independent storage under its type constraints.

The fixed im and supervise implementations obey reviewed API contracts.
UncheckedValue is the explicit package-level immutability promise. Do not expose
its payload through a local projection or add another trust annotation.

Every package with user-declared Frozen records must have this directive:
`//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen`.
Declare Frozen records as named package-scope types in non-test Go files.
Reject function-local, anonymous, and test-only Frozen declarations.
Keep the exact current `zz_immutago.gen.go` output. Frozen records get typed
projections and Cells. Local and Copied records need no generation. Keep the two
generated-only operations above separate from the normal checks; do not skip
analysis of a generated file.

Test supported and rejected forms through the real CLI with diagnostic snapshots.
Keep one `output.txt` per fixture package for combined stdout and stderr. Keep
source and generated Go files separate; do not add `.gen.txt` companions.
Check runtime representations and allocation costs. Keep ordinary Go outside
protected operations unaffected.

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

Use the [language reference](docs/rules.md) for the complete supported forms.
Keep source restrictions and examples there. Keep analysis details in the
[architecture pages](docs/arch/README.md).

### Analysis requirements

- Apply builder ownership checks when protected elements or a later Freeze
  require them. A scalar Clone that is never frozen follows ordinary Go rules.
- Reuse completed type proofs and cached Cell queries. Test cycles and query
  order. Use the existing cache during recursive walks.
- Keep one real failure cause for each invalid type. Test correct rejection and
  terminating cause paths. The selected cause may depend on query order.
- Carry one representative original failure in each dependency fact, with a
  command to check its origin.
- Walk each storage selector/index chain once. Keep one storage classifier and
  one Cell/source context.
- Pass the builder owner ID and consume/reset flags directly to the bitset solver.
- Allow no result or one ReadResult from each exported service method
  except Init and Run. Runtime methods are exempt. Other service code follows
  ordinary Go rules; im operations use their general checks.
- Permit service references as stored values and results. Match every direct Ref Borrow
  call as `ref.Borrow().Method(args)` with an exported service method other
  than Init, Run, or a runtime method. Reject all other direct uses of the returned
  pointer. Interface conversion is allowed and can expose the service pointer. Keep the reference proof in [service architecture](docs/arch/services.md).
- Trust ordinary Go API contracts. Use the documented package selection rules;
  do not add string taint analysis or whole-program service effect analysis.

Require a reproduction and a relevant before/after measurement for a performance
claim. Keep measurements in the PR, not in user guides. A smaller diff alone does
not prove a better algorithm.

### Compiler-constant operands

Apply the same source rules to all operands, including operands that Go does
not evaluate. Constant operands do not bypass capture, type placement, generic,
or Freeze checks. Direct array `len` and `cap` work in allowed statements.

For a local `p *Local`, where Local embeds `im.Local`:

| Rejected form | Allowed replacement |
| --- | --- |
| `f := func() int { return len([1]*Local{p}) }` | `f := func() int { return 1 }` |

An ordinary length computed before the closure can also be captured. Keep the
reference, accepted/rejected fixtures, and matcher aligned with this rule.
