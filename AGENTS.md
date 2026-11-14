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
not source spelling. Do not accept unknown cases or add escape hatches.
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
