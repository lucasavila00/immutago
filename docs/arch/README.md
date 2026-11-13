# Architecture

Immutago adds checks to Go; it does not change the Go compiler or runtime.
The intended contract is a small ownership system: local values can change,
shared references permit reads, and freezing consumes an exclusive native
container. The explicit subset rejects the source forms from the two confirmed review
failures. Tests are evidence, not a formal proof that all defects are absent;
see [validation and review regressions](validation.md).

These notes describe the explicit subset and its fixed-owner loan analysis.

- [Analysis pipeline](pipeline.md): compiler types, structural checks, operation
  checks, and dependency facts.
- [Ownership and storage](ownership.md): value copies, frozen containers,
  control-flow graphs, and local loans.
- [Validation and review regressions](validation.md): trusted code, tests, review findings,
  and limits.

Read the [intended contract](../guide/immutability.md) for accepted source forms.
The [borrowing ADR](../adr/borrowing/what.md) and
[frozen collections ADR](../adr/frozen-collections/what.md) explain the design
choices. The architecture notes distinguish those choices from demonstrated
implementation behavior.
