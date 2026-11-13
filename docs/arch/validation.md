# Validation

[Architecture](README.md) · [Reviewer requirements](../reviewer-focus.md)

## Trusted code

The [analyzer](../../pkg/analyzer/analyzer.go) skips the fixed `im` package path
and the standard library. Toolchain detection
identifies standard library source from the active `go env GOROOT` directory.
The implementation trusts the Go compiler, runtime, standard library, and
`im` storage methods. It does not prove those implementations.

Checked application code cannot use reflection or unsafe imports to bypass
ownership. An unsupported operation produces an error. There are no user trust
annotations or suppression directives.

The analyzer must run for the consumer's packages and build configuration.
An ordinary Go build does not run it. The analysis driver checks dependencies
and carries relevant failures to selected consumers. Freeze itself has no
alias check, consumed flag, or element validation at runtime. Code used without
the analyzer does not receive these guarantees.

This is a static checker with regression evidence, not a formal verification
of its implementation. A written allowlist alone does not prove that every
source form reaches the correct check.

## Evidence by layer

| Layer | Evidence |
| --- | --- |
| Recursive type proofs | [Type tests](../../internal/checker/types_test.go) cover recursive field graphs and marker modes. |
| Allowed source forms | [Analyzer tests](../../pkg/analyzer/explicit_subset_test.go) and [shape tests](../../pkg/analyzer/shape_convergence_test.go) check complete forms and nearby rejections. |
| Builder ownership | [Freeze tests](../../internal/freeze/freeze_test.go) check fresh storage, aliases, branches, loops, and consumption. |
| Library storage | [Slice tests](../../pkg/im/slice_test.go), [map tests](../../pkg/im/map_test.go), and [iterator tests](../../pkg/im/iterator_test.go) check storage reuse, clone independence, reads, and allocations. |
| Shipped analyzer | [CLI tests](../../tests/cli_test.go) build the command and load fixtures as independent consumer modules. |

CLI tests cover compiler errors, analyzer output, JSON, and `go vet`.
Dependency tests select only consumers, so failures hidden by a driver must
still reach them. Sibling `.txt` snapshots record source positions, context,
and corrections. Use the [snapshot workflow](../development.md#diagnostic-snapshots)
to inspect changes before accepting them.

[Hosted CI](../../.github/workflows/ci.yml) runs source checks, formatting,
generated-doc checks, a build, and repository self-analysis. Race-enabled tests
run on Linux, macOS, and Windows. Use local runs for focused debugging and
hosted CI for full validation.

## Required regression coverage

Keep accepted and rejected cases for each supported form:

| Area | Accepted cases | Rejected nearby cases |
| --- | --- | --- |
| Immutable objects | Scalar preparation, whole-variable rebinding, copies, value and pointer constructors | Field writes, protected array writes, dereference writes, named-variable addresses |
| Shared references | Copy, reassignment, stored pointers, returns, recursive pointer fields | Scalar interior addresses, pointer indirection, interfaces, captures, generic protected code, asynchronous calls |
| Statement forms | One destination, fresh map-reader results, ordinary loops, separate Freeze | Nested protected operands in multiple destinations, nested Freeze, protected select communication |
| Builders | Fresh construction, element edits, nil and empty storage, fresh loop declarations | Aliases before Freeze, subslices, element addresses, unknown or external sources, active-range Freeze |
| Consumption | Frozen reads, Clone, independent edits, refreezing the clone | Reads or writes after Freeze, double Freeze, indirect or asynchronous Freeze |
| Types and calls | Recursive immutable elements, exact collection methods, direct named calls | Invalid elements, user methods on protected receivers, dynamic protected results, callback transport |
| Dependencies | Checked readers in imported packages | Dependency failures hidden by consumer-only CLI or vet runs |

For reference transport, test returns, fields, globals, containers, scalar
addresses, pointer indirection, conversions, interfaces, captures, generics,
`defer`, and goroutines separately. Test parentheses, aliases, implicit
receivers, iterator bindings, and tuple results around each affected form.

Runtime tests must check that Freeze reuses original storage and that Clone
creates independent storage. Allocation tests separate native construction
from Freeze, reads, and direct iteration. See [collection costs](../performance.md) and
[reviewer requirements](../reviewer-focus.md).

## Regression cases to retain

These cases explain why complete source forms matter. The linked fixtures and
snapshots record their current behavior.

| Case | Required check | Fixture |
| --- | --- | --- |
| A promoted scalar method writes through a protected owner | Reject user-defined methods on protected receivers, including implicit receivers. | [Protected methods](../../tests/testdata/rejected/explicit_methods/explicit_methods.go) |
| A select operand freezes a builder before a selected receive writes into it | Reject protected select communication operands and targets. | [Statement forms](../../tests/testdata/rejected/explicit_statements/explicit_statements.go) |
| A protected read is nested inside a multiple assignment or range target | Match the complete enclosing statement, not only its top-level operands. | [Rejected forms](../../tests/testdata/rejected/review_convergence/convergence_rejected.go) |
| A callback returns a protected value without taking one | Check result types as well as arguments. | [Rejected calls](../../tests/testdata/rejected/review_convergence/convergence_rejected.go) |
| Parentheses or a shared receiver hide a valid collection operation | Resolve exact method identity and allow the documented syntax. | [Accepted forms](../../tests/testdata/accepted/review_convergence/convergence_allowed.go) |

Do not repair a regression by adding only its spelling to a forbidden list.
Match the allowed form and reject its alternatives. Keep an accepted nearby
example so the repair does not remove a documented operation.

## Scope

The checker uses recursive type graphs, complete source-form checks, exclusive
builder checks, and dependency facts. It has no loan lifetimes or mutable
references. Go manages shared object lifetime. See the [rules](../rules.md) for
the complete supported forms and [storage analysis](ownership.md) for the
implementation.
