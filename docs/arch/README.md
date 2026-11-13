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

Read the [guide](../guide/immutability.md) for accepted source forms and the
[project reference](../README.md) for API, cost, and development documentation.

## Design choices

- Value constructors return `T`. This avoids an owning-pointer model such as
  Rust's `Box<T>`.
- Each local reference has one fixed owner. This removes the forward alias
  solver while retaining backward NLL for branches and loops.
- A loan protects the whole owner. This avoids field-overlap analysis, but
  rejects some disjoint field writes that Rust permits.
- Direct named readers cannot retain references. This avoids dynamic-call
  effects and pointer-return summaries across packages.
- Freeze transfers a proved exclusive native handle. The small `im` wrappers
  are trusted primitives; the checker does not prove arbitrary storage
  libraries with native pointer fields.
- Clone creates independent native storage for edits. There are no persistent
  update trees or mutable borrows.

The [ownership page](ownership.md) explains these mechanisms. The
[validation page](validation.md) states what is trusted and what the tests show.
