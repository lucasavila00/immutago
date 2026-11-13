# Reviewer requirements

[Documentation](README.md) · [Language rules](rules.md) · [Validation](arch/validation.md)

Review every checker or supported-language change against these two requirements.

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

Check `Shared` sharing and `LocalRef` reference escape separately. Builder
exclusivity and consumption remain required for both modes.

## Required evidence

For each changed form, identify:

1. Its guide entry and Go example.
2. Its complete matcher and rejecting fallback.
3. Its type, write, escape, or builder proof checks.
4. Accepted CLI fixtures and nearby rejected forms, with exact snapshots.
5. Dependency coverage when the form crosses package boundaries.

Check implicit receivers, new bindings, aliases, parentheses, tuple results,
and statement evaluation order. Preserve ordinary Go outside protected storage.
A forbidden form passing is a defect even if the guide does not show that
rejection example. Known negative tests do not replace the matcher review.

Check changed links and generated package references. Use hosted CI for full
validation. For collection changes, also check the [cost contract](performance.md).
