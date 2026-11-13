# Rust compiler algorithm sources

Backward borrow liveness is translated from the Rust compiler at commit
`80196be0e787008b6fb14d2990aba92083cd68e1`. The translated code uses the upstream
MIT license in [LICENSE-MIT](LICENSE-MIT).

| Source | Algorithm | Go implementation |
| --- | --- | --- |
| [`type_check/liveness/trace.rs`](https://github.com/rust-lang/rust/blob/80196be0e787008b6fb14d2990aba92083cd68e1/compiler/rustc_borrowck/src/type_check/liveness/trace.rs) | Backward use-to-definition liveness | `internal/borrow/flow/liveness.go` |
| [`rustc_mir_dataflow/framework/mod.rs`](https://github.com/rust-lang/rust/blob/80196be0e787008b6fb14d2990aba92083cd68e1/compiler/rustc_mir_dataflow/src/framework/mod.rs) | Reverse-postorder traversal for fixed-point iteration | `internal/borrow/flow/graph.go` |

The Go adapter uses typed syntax and statement control flow. It separates reads,
assignment writes, and reference-binding definitions to preserve evaluation
order within the supported source forms. Each local reference declaration
records one fixed whole owner. Reference copying and rebinding are rejected.
Backward liveness determines whether a reference still matters at a write; no
forward possible-owner fixed point remains.

Whole-owner loans and non-escaping readers remove the need for field-overlap
rules, outlives-region propagation, returned-alias summaries, and recursive
pointer-path widening. Those algorithms are not part of this implementation.
Go keeps its zero initialization, value copies, and garbage collector; the
checker does not implement Rust's move or named-lifetime systems.
