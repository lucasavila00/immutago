# Rust compiler algorithm sources

The borrow-analysis algorithms are translated from the Rust compiler at
commit `80196be0e787008b6fb14d2990aba92083cd68e1`. Translated code is used under
the upstream MIT license in [LICENSE-MIT](LICENSE-MIT).

| Source | Algorithm | Go implementation |
| --- | --- | --- |
| [`type_check/liveness/trace.rs`](https://github.com/rust-lang/rust/blob/80196be0e787008b6fb14d2990aba92083cd68e1/compiler/rustc_borrowck/src/type_check/liveness/trace.rs) | Backward use-to-definition liveness | `internal/borrow/flow/liveness.go` |
| [`region_infer/region_context.rs`](https://github.com/rust-lang/rust/blob/80196be0e787008b6fb14d2990aba92083cd68e1/compiler/rustc_borrowck/src/region_infer/region_context.rs) | SCC outlives propagation | `internal/borrow/flow/regions.go` |
| [`rustc_mir_dataflow/framework/mod.rs`](https://github.com/rust-lang/rust/blob/80196be0e787008b6fb14d2990aba92083cd68e1/compiler/rustc_mir_dataflow/src/framework/mod.rs) | Reverse-postorder, minimum-rank dirty worklist | `internal/borrow/flow/graph.go` |
| [`dataflow.rs`](https://github.com/rust-lang/rust/blob/80196be0e787008b6fb14d2990aba92083cd68e1/compiler/rustc_borrowck/src/dataflow.rs) | First out-of-region kills and forward loan transfer | `internal/borrow/flow/liveness.go` |

The Go lowering must preserve source storage identities, evaluation order, and
pointer provenance. It maps normalized operation points to the graph engines.
The Rust algorithms use MIR blocks and compact bitsets; the Go port represents
individual operations and integer sets. Go's zero initialization, garbage
collection, and reference-copy behavior are handled by the Go adapter.
