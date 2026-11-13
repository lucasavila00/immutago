# Borrowing: WHAT

## Decision

Use Rust-style ownership and shared loans for marked Go values. Borrow lifetimes
end at their last required use.

Owned local storage permits conditional construction, field updates, and
reassignment. A value copy has independent storage. A pointer copy aliases its
target. Shared pointers permit reads, but no writes through the pointer.

```go
var config Config // Config embeds im.Marker
if enabled { config.Limit = 10 } else { config.Limit = 2 }
p := &config
read(p)
config.Limit = 20 // accepted if read neither retains nor returns p
```

Writing `config.Limit` before the last required use of `p` fails. Replacing
`config` also fails while its storage is borrowed. Rebinding `p` to a different
allocation does not replace its previous target.

Fresh owned allocations support construction before sharing. Returned, captured, and passed references retain their loans. Unknown effects fail.

Fields and elements recursively allow primitives, arrays, marked structs, and
`im` collections. Custom aggregates must be marked structs; defined array
wrappers are rejected.
Unmarked structs cannot contain immutable fields. Published collection storage
is never writable. Package-level marked storage is read-only after initialization.
No suppression directives or unchecked user implementations are permitted.
