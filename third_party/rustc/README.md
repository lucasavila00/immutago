# Rust compiler algorithm sources

The graph traversal in `internal/freeze/flow` uses the Rust compiler at commit
`80196be0e787008b6fb14d2990aba92083cd68e1`. The translated code uses the upstream
MIT license in [LICENSE-MIT](LICENSE-MIT).

| Rust source | Algorithm used |
| --- | --- |
| [`rustc_mir_dataflow/framework/mod.rs`](https://github.com/rust-lang/rust/blob/80196be0e787008b6fb14d2990aba92083cd68e1/compiler/rustc_mir_dataflow/src/framework/mod.rs) | Reverse-postorder traversal for fixed-point iteration |

This directory records source attribution and contains the upstream MIT license.
Only the graph traversal listed above is adapted. Builder consumption uses a
forward dataflow check; object references do not require lifetime analysis.

For their use in Immutago, see [ownership analysis](../../docs/arch/ownership.md).
