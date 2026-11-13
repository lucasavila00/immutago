# Borrowing: REVIEW

## Reviewed rules

The supported Rust subset is `T`, `&T`, and NLL. Owned values can change. Shared
references cannot write. An owner can change again after its last shared borrow
ends. Go copies values instead of moving them.

The implementation was compared with the pinned rustc algorithms. A write is
checked against live loans before overwrite kills apply. Possible overlap is
used to reject conflicts; definite overlap is required to kill a loan.

## Review findings

The review found and corrected lost aliases through helpers, returned globals,
loop iterations, pointer indirection, and copied analysis state. Scalar pointers
into marked structs retain the same protection as pointers to the whole struct.

Recursive pointer paths use a finite conservative representation. An unknown
path cannot justify a write or end a borrow. Unknown effects must follow pointer
slots to the storage they reference.

## Evidence

Targeted debugging covered owned construction, copies, constructors, last-use
writes, and the rejection cases above. Independent reference solvers check the
ported graph operations. End-to-end tests exercise the CLI and `go vet`.

Hosted CI supplies the final cross-platform, race, lint, documentation, build,
and repository checks. Passing a narrow test does not replace those checks.
