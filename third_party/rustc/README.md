# Graph traversal attribution

[internal/freeze/flow/graph.go](../../internal/freeze/flow/graph.go) adapts
reverse-postorder traversal from Rust compiler commit
`80196be0e787008b6fb14d2990aba92083cd68e1`:
[`rustc_mir_dataflow/src/framework/mod.rs`](https://github.com/rust-lang/rust/blob/80196be0e787008b6fb14d2990aba92083cd68e1/compiler/rustc_mir_dataflow/src/framework/mod.rs).
The adapted code uses the upstream [MIT license](LICENSE-MIT).

The traversal orders Immutago's [builder consumption solver](../../docs/arch/ownership.md).
This attribution covers that graph algorithm; object references need no
lifetime analysis.
