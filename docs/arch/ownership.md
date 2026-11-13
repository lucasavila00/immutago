# Ownership and storage

[Architecture index](README.md)

## Values and frozen storage

Go copies structs and arrays by value. Marked fields and protected array elements cannot change after
construction. A complete variable can be rebound to a new value. A value copy has separate struct and array storage.
Frozen collection fields still share their backing storage after a value copy;
the checker requires that storage to remain read-only.

[`FrozenSlice`](../../pkg/im/slice.go) and
[`FrozenMap`](../../pkg/im/map.go) contain private native handles. Freeze copies
the handle into the wrapper. It does not copy elements or enforce ownership at
runtime. `Len`, `At`, and `Get` read values. [`All`](../../pkg/im/iterator.go) returns
an iterator that reads the same backing storage. `Clone` copies native storage
in O(n) time and space. The copy is shallow; valid element types must make that
sufficient. There are no persistent update trees or path-copy updates.

The freeze operation resembles a Rust move, but Go does not perform
that move. The analyzer must reject every prohibited alias and later builder
use. The runtime wrapper alone cannot supply this guarantee.

## Builder analysis

[Discovery](../../internal/freeze/discover.go) indexes AST parents and local
container declarations by `*types.Var`. A builder must start with `make`, a
literal, nil, a zero declaration, or a recognized `Clone` call. Discovery selects
native containers with protected elements and inputs to direct freeze calls.

[Use checks](../../internal/freeze/uses.go) reject handle aliases and operations
outside the builder subset. They examine uses before freezing too: an alias
created earlier would survive the transfer. Unknown helper calls, subslices,
element addresses, and escaping storage cannot establish exclusive ownership.

[Flow checks](../../internal/freeze/flow.go) build a Go control-flow graph for
each participating function, then expand blocks into use, consume, and reset
events. Each builder has an integer identity. Sets of consumed identities flow
forward. A join takes the union of predecessor states; consumption on any
incoming path therefore prevents a later use. A fresh declaration resets its
identity, including when that declaration runs again inside a loop. Freeze is
rejected while a range over the same builder is active: the range still retains
its native handle. Direct synchronous Freeze calls make consumption explicit;
function-value aliases cannot hide that operation.

The solver scans reachable points in reverse postorder until output sets stop
changing. Input and output sets are maps. Built-in writes and freeze effects
are recorded after their operands have been evaluated.

## Immutable object checks

The source matcher permits fresh protected pointer construction with a composite
literal or `new`. It rejects addresses of named variables and their fields.
This prevents variable rebinding from changing a shared object's storage.

The matcher permits reads, value copies, pointer copies, pointer reassignment,
and concrete typed storage and returns for `Immut`. References to
`ImmutNoRef` cannot be stored, returned, or captured; direct typed calls must
not retain them. Value copies remain storable and returnable. Write destinations must be complete
variables or supported native builder elements. A marked field, protected array
element, or dereferenced pointer is never a valid write destination. This rule
holds before sharing and after the last read; no loan analysis is needed.

The recursive type proof follows protected pointer targets. Every reachable
field must pass, including fields reached through recursive generic types.
Scalar pointer fields and pointers to pointers do not pass. Collections do not
expose addresses into their backing storage.

The shared [graph code](../../internal/freeze/flow/graph.go) stores predecessor
and successor lists for builder consumption. Its
[sets](../../internal/freeze/flow/set.go) are `map[int]bool`. The
[source and license notes](../../third_party/rustc/README.md) identify adapted
graph code. There is no borrow checker or MIR pipeline.
