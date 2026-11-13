# Reviewer requirements

[Documentation](README.md) · [Validation](arch/validation.md)

Use this checklist for every change to the supported Go subset. The
[guide](rules.md) defines the allowed source forms. The
[architecture notes](arch/README.md) explain how they are checked.

## 1. Document what is allowed

The allowed-form list must be complete for the operation being described.
Each entry must include a short Go example. State that all other forms are
rejected. Use concrete rejected examples and allowed replacements; state that
those rejection examples are not exhaustive.

For example:

```go
// Allowed: construct a fresh immutable object.
p := &Point{X: 1}

// Rejected: address a variable that can be rebound.
q := &value

// Allowed replacement: share the existing object.
q := p
```

Do not substitute phrases such as "keep aliases visible" for an exact form.
Check the source examples against the real CLI, including imports and types.
A claim that a list is complete requires evidence from the matcher and tests.

## 2. Match allowed shapes; reject the rest

The implementation must match the complete allowed form, like function clauses
that match specific argument shapes and finish with a rejecting fallback.

For fresh pointer construction, match all of these parts:

- One fresh composite literal or supported `new` expression.
- A resolved pointer to a recursively proved protected type.
- An allowed expression and statement context.

Use the [reference rules](rules.md#what-you-can-do-with-a-reference) to check
construction, copying, storage, returns, nil, and reassignment separately.
Addresses of named values and their fields must reach the rejecting path.
Do not accept every address expression and add exceptions after bugs appear.
Use compiler identities, not only names such as `Get` or `FreezeSlice`.

Check every write destination. Complete variable rebinding must not admit a
field, protected array element, or pointer-target write. Check implicit method
receivers, statement context, and Go evaluation order. Builder consumption
must remain separate from immutable object sharing.

## Required review evidence

For each changed form, identify:

1. Its allowed entry and example in the guide.
2. The matcher that accepts it and the fallback that rejects other forms.
3. The ownership checks applied after the match.
4. Accepted CLI fixtures and rejected nearby forms, with sibling snapshots.
5. Dependency checks when the behavior crosses package boundaries.

Test new bindings as well as ordinary uses. Iterator bindings and type-switch
bindings must not bypass checks because they are absent from assignment nodes.
Test implicit operations, parentheses, type aliases, and extra results. Preserve
ordinary Go operations that do not involve protected storage.

A new supported form requires docs, a matcher, ownership rules, and tests in the
same change. Do not call a gap fixed by changing only the docs, only a rejection
example, or only one spelling of an unsupported operation.

## Review documentation changes

Check both reading paths from the documentation index. A user must be able to
install the tool, run a complete example, find an allowed form, and understand
a diagnostic. A maintainer must be able to find the check, its invariant, its
tests, and the validation command.

Check claims against resolved type checks, source-form matches, ownership code,
and fixtures. Keep rule definitions on the rules page; architecture pages
explain the mechanism and link to the rule. Remove stale future-tense plans and
avoid repeating historical reports as current behavior. Keep temporary
investigation reports in the change history, not in the reference pages.

Run complete new examples through the CLI. Check relative links and anchors
after moving pages, and check template paths if documentation tooling moves.
Generated references must still match their Go comments.

For collection performance changes, compare equivalent native Go and `im` code.
Check inlining and heap escapes as well as allocations and elapsed time. The
[zero-overhead target](performance.md#zero-abstraction-overhead) is stronger than
having no runtime ownership checks.
