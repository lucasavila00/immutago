# Rust compiler algorithm sources

Backward borrow liveness is translated from the Rust compiler at commit
`80196be0e787008b6fb14d2990aba92083cd68e1`. The translated code uses the upstream
MIT license in [LICENSE-MIT](LICENSE-MIT).

| Rust source | Algorithm used |
| --- | --- |
| [`type_check/liveness/trace.rs`](https://github.com/rust-lang/rust/blob/80196be0e787008b6fb14d2990aba92083cd68e1/compiler/rustc_borrowck/src/type_check/liveness/trace.rs) | Backward use-to-definition liveness |
| [`rustc_mir_dataflow/framework/mod.rs`](https://github.com/rust-lang/rust/blob/80196be0e787008b6fb14d2990aba92083cd68e1/compiler/rustc_mir_dataflow/src/framework/mod.rs) | Reverse-postorder traversal for fixed-point iteration |

This directory records source attribution and contains the upstream MIT license.
Only the algorithms listed above are adapted; this is not a copy of the full
Rust borrow checker.

For their use in Immutago, see [ownership analysis](../../docs/arch/ownership.md).
