# Reviewer requirements

Review every checker or supported-language change against these two requirements.

## Local review

- State the rule's checks and the facts a reviewer can rely on.
- Review service state changes, locks, saved inputs, and dependency calls in the
  service code. Ordinary Go contracts apply.
- Do not claim full service isolation from immutable result types.

## 1. Complete allowed forms

The guide must list every allowed Go form with a short example. It must say
that unlisted operations are rejected. Rejection examples must include allowed
replacements and must be identified as non-exhaustive.

Check concrete construction, assignment, parameter, result, receiver, and
statement shapes. Intent such as “keep aliases visible” is not a source rule.
Run complete examples through the real CLI.

## 2. Complete matches with a rejecting fallback

The checker must match the whole documented shape using AST nodes and resolved
Go type identity, then apply its proof checks. Unknown forms must fail. Do not
accept a broad syntax class and add exceptions after defects are found.

For example, fresh protected construction requires `&T{...}` or a supported
`new` form, a concrete marked target, and an allowed statement context.
`&variable` and `&value.Field` must fail. Whole-variable replacement must not
admit field, element, or pointer-target writes.

Check `Shared` sharing and `ByValue` pointer escape separately. Builder
exclusivity and consumption are required for both modes.

For local frozen views, trace the complete containing type through fields and
arrays. Check package storage, Owner, service arguments, collection entries,
direct calls, and results. Check data returned by At, Get, and All, not only
the view handle. Confirm the text-key and nested Clone restrictions against
the [local view lists](local-views.md).

## Required evidence

For each changed form, identify:

1. Its guide entry and Go example.
2. Its complete matcher and rejecting fallback.
3. Its type, write, escape, or builder proof checks.
4. Accepted CLI fixtures and nearby rejected forms, with exact snapshots.
5. Dependency coverage when the form crosses package boundaries.

Check implicit conversions and receivers, actual source and destination types,
new bindings, aliases, parentheses, tuple results, variadic elements, and statement
evaluation order. Preserve ordinary Go outside protected storage.
A forbidden form passing is a defect even if the guide does not show that
rejection example. Known negative tests do not replace the matcher review.

For each defect, identify why the operation missed a matcher or passed an
incomplete match. Check the same path through declarations, nil and zero values,
aliases, nested storage, tuples, and variadic arguments where relevant. Fix that
path and test both allowed and rejected forms. A test for one failing example
alone is not sufficient.

Check changed links and generated package references. Use hosted CI for full
validation. For collection changes, also check the [cost contract](performance.md).

## Public guide review

Read as a Go developer who has not worked on the checker:

- Name the API before discussing it: `im.Shared` and `im.ByValue`, not an unexplained
  “both markers.” Define terms before using them.
- Use the [glossary](glossary.md) names. Distinguish ByValue values from ByValue
  pointers, and service references from Go pointers. A service result is a
  result category, not a marker or a promise of full service isolation.
- Show what the feature does, when to choose it, and a useful working example.
- Use lists for rules, tables for comparisons, and numbered steps for sequences.
- Remove repeated prose; retain facts needed to use the example correctly.
- Keep complete allowed forms in the reference and checker mechanics in architecture.
- Check reading order, links, generated references, and examples through the CLI.

Passing examples does not make a guide clear. Review the explanation too.
