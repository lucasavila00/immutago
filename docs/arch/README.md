# Architecture

Immutago adds checks to Go; it does not change the Go compiler or runtime.
The intended contract is a small ownership system: local values can change,
shared references permit reads, and freezing consumes an exclusive native
container. The current implementation has known gaps. A successful check is
not a complete proof of the contract; see [validation and gaps](validation.md).

These notes describe the source at `0d7e423`. They explain the current design,
not a proposed replacement.

- [Analysis pipeline](pipeline.md): compiler types, structural checks, operation
  checks, and dependency facts.
- [Ownership and storage](ownership.md): value copies, frozen containers,
  control-flow graphs, and local loans.
- [Validation and gaps](validation.md): trusted code, tests, review findings,
  and limits.

Read the [intended contract](../guide/immutability.md) for accepted source forms.
The [borrowing ADR](../adr/borrowing/what.md) and
[frozen collections ADR](../adr/frozen-collections/what.md) explain the design
choices. The architecture notes distinguish those choices from demonstrated
implementation behavior.
