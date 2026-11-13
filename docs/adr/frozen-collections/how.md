# Frozen collections: HOW

Wrap a native slice or map handle in an unexported field. Read methods return
value copies. Clone copies native storage. No update method or writable view
exposes frozen storage.

Before each Freeze call, prove a fresh local source: `make`, a literal, nil,
or the matching frozen type's Clone. Reject handle aliases, subslices, element
addresses, closures, unknown calls, returns, and storage outside the local
binding. Check uses before freezing as well as after it.

Use the Go control-flow graph to propagate consumed state. Any path that freezes
a builder makes a later use invalid. A fresh declaration inside a loop creates
new ownership on each iteration. Freezing during an active range is rejected.

Only direct synchronous Freeze calls are allowed. Reject function-value aliases
so an indirect call cannot hide ownership consumption. Imported dependency proof
failures propagate through the existing analyzer facts.

The proof has no runtime representation. Ordinary Go allocates the original
backing storage; Freeze only copies its handle. Clone deliberately costs O(n).
