# Ownership and storage

[Architecture index](README.md)

## Values and frozen storage

Go copies structs and arrays by value. A marked local can change until a shared
loan prevents a write. A value copy has separate struct and array storage.
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

## Shared-loan analysis

[Borrow analysis](../../internal/borrow/local.go) handles one relevant function
body at a time. [Lowering](../../internal/borrow/lower.go) starts with
`golang.org/x/tools/go/cfg`, then creates operation points for reads, writes,
and pointer binding definitions. The lowering evaluates operands
before a single assignment effect. Protected multiple assignments and select
communication operations are rejected by the source-shape checks.

A place such as `shape.Child` or `items[i]` reduces to its whole owner variable.
The analysis does not distinguish disjoint fields or array elements. This
keeps overlap checks small but rejects some programs that Rust could accept.

[Lowering](../../internal/borrow/lower.go) records one fixed owner for each
local reference. The source matcher permits only an address-initialized local
binding or a named reader parameter. Copies and reassignment are rejected.
There are no per-operation possible-owner maps or forward owner propagation.

[Backward liveness](../../internal/borrow/solve.go) starts at pointer uses and follows predecessor edges until
definitions. Each local reference has a set of live operation points. A reference
can stop protecting its owner after its last required use, before lexical scope
ends. Reader parameters need no local owner entry: their resolved pointer types
already prohibit writes through them.

Writes through shared references fail. A write to an owned value fails when a
live pointer can refer to that owner. Package-level protected storage cannot
change after declaration initialization. The [validation notes](validation.md)
link source-form regressions to their current tests.

The shared [graph code](../../internal/borrow/flow/graph.go) stores predecessor
and successor adjacency lists. Its [sets](../../internal/borrow/flow/set.go)
are `map[int]bool`. The backward traversal and reverse-postorder approach are
adapted from rustc; [source and license notes](../../third_party/rustc/README.md)
identify the source. This is not rustc's full borrow checker or MIR pipeline.
