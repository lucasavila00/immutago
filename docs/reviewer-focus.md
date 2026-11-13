# Reviewer requirements

Use this checklist for every change to the supported Go subset. The
[guide](guide/immutability.md) defines the allowed source forms. The
[architecture notes](arch/README.md) explain how they are checked.

## 1. Document what is allowed

The allowed-form list must be complete for the operation being described.
Each entry must include a short Go example. State that all other forms are
rejected. Use concrete rejected examples and allowed replacements; state that
those rejection examples are not exhaustive.

For example:

```go
// Allowed: one new reference from an owned local.
p := &value

// Rejected: copying a reference.
q := p

// Allowed replacement: borrow the owner directly.
q := &value
```

Do not substitute phrases such as "keep aliases visible" for an exact form.
Check the source examples against the real CLI, including imports and types.
A claim that a list is complete requires evidence from the matcher and tests.

## 2. Match allowed shapes; reject the rest

The implementation must match the complete allowed form, like function clauses
that match specific argument shapes and finish with a rejecting fallback.

For a local reference, match all of these parts:

- One new local binding.
- One address expression as its initializer.
- An owned local value or its marked child as the address target.
- A resolved protected pointer type.
- An allowed statement context.

Only after that match may the loan analysis record the owner. A declaration from
an iterator, a pointer copy, a nil initializer, or a reassignment must reach the
rejecting path. Do not accept every declaration and add exceptions after bugs
appear. Use compiler identities, not only names such as `Get` or `FreezeSlice`.

Check each allowed method's actual receiver and each statement's Go evaluation
order. Source syntax can hide an implicit address or a delayed write. A correct
liveness algorithm does not repair an incorrect input operation graph.

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
