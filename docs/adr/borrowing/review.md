# Borrowing: REVIEW

## Reviewed contract

Shared pointer parameters are read-only. A factory can construct its own fresh
storage and return it; this does not grant permission to mutate a caller's shared
parameter. By-value helpers receive independent storage. A second pointer to
owned storage creates a shared loan; the owner can write again after that loan's
last use. Pointer rebinding changes the binding, not the previous target.

## Port requirements

rustc uses possible overlap for conflicts and definite
overlap for loan kills. Do not interchange these predicates. Check a write
against live loans before its overwrite kills. Propagate region uses
through joins, loop back edges, aliases, and returned references.

Go lowering must preserve operand evaluation before assignment stores. Track
implicit pointer-receiver method borrows and interior scalar pointers. Deferred
arguments remain live until the deferred call; closure captures and goroutines
cannot be treated as immediate calls.

## Acceptance gates

Tests must distinguish owner storage, independent copies, and shared references.
Require branch, loop, reborrow, factory, generic, cross-package, and early-loan-end
cases. Unknown call effects must fail explicitly. Passing collection tests or
blanket pointer rejection does not validate the borrow engine. Review the
implementation and recorded proof results against these gates before completion.
