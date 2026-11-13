# Frozen collections: WHAT

Replace persistent collections with `im.FrozenSlice[T]` and
`im.FrozenMap[K, V]`. `FreezeSlice` and `FreezeMap` consume exclusive native
storage without copying or allocating. Keep read operations and `Clone`;
remove constructors that copy inputs and persistent update methods.

```go
values := []int{1, 2}
frozen := im.FreezeSlice(values)
copy := frozen.Clone()
copy[0] = 3
next := im.FreezeSlice(copy)
```

The original frozen value remains unchanged. The checker rejects later uses of
consumed native handles and aliases that could retain writable access. Keys
and elements must recursively be immutable. Frozen wrapper copies share storage.

This is a checked ownership transition, like moving a Rust builder into an
immutable owner. It is not a runtime lock or a mutable-borrow extension.

The small `im` wrappers remain trusted runtime primitives. The checker recognizes
their exact package/type identities, checks callers and type arguments, and has
no user escape hatch. This decision does not claim to prove arbitrary library
implementations with native reference fields.
