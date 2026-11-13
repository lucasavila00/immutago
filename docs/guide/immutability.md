# Immutability and borrowing

Embed `im.Marker` directly by value. Every field must recursively be a primitive
boolean, number, or string; an array of allowed elements; another marked struct;
or `im.Slice[T]` / `im.Map[K, V]` with allowed arguments. Type aliases retain their
identity. Custom aggregates must be marked structs. For example, replace
`type Count [2]int` with an inline array in a marked struct:

```go
import "github.com/lucasavila00/immutago/pkg/im"

type Count struct {
    im.Marker
    Values [2]int
}

type User struct {
    im.Marker
    Name string
    Tags im.Slice[string]
}
```

Owned local storage permits ordinary Go construction, including conditionals,
loops, field updates, and whole-value reassignment. Go zero values are valid
defaults. A value copy owns separate storage; its immutable collection fields
can safely share their library representation. Value-receiver copy-and-update
methods are supported.

```go
var user User
if admin {
    user.Name = "Admin"
} else {
    user.Name = "Guest"
}
user.Tags = im.NewSlice("go")
p := &user
name := p.Name        // last use of this reference
user.Name = "Reader" // accepted: the loan has ended
_ = name
```

Taking a reference to existing storage creates a shared loan. Reading through
that reference is allowed. Writing through it is not. Writing or replacing its
owner fails while a conflicting loan is live. Pointer copies retain the loan;
reassigning one pointer does not remove loans held by other aliases. Loans end
at their last required control-flow use, including future loop iterations.
Borrowed fields retain their source ownership even when the pointer type is
only `*int`. Nonoverlapping fields and constant array indices are distinguished.

Fresh allocations such as `p := &User{}` and `new(User)` retain an owner
capability for construction. Copying that owner into another pointer creates a
shared alias. A checked factory can return a fresh owner; a function returning
an input pointer preserves that input's shared provenance. Incoming pointers
and native backing storage are read-only when they refer to marked values.

Calls use inferred effects and returned-alias summaries, also across packages.
A read-only call does not extend a loan after its last required use. Deferred
reads remain live until execution; captured references cannot be treated as dead
at the capture statement. Unproved relevant call effects fail explicitly.
Package-level marked storage is read-only after declaration initialization.

`Slice` and `Map` updates return persistent snapshots. Inputs and `Values()`
outputs cross mutable boundaries by copying. `All()` iterates without making a
native container copy. Proof checks run only in the analyzer; collection costs
are described in [performance](../performance.md).

# Boundaries

Unmarked structs cannot contain marked values or `im` collections. Marked fields
and collection arguments cannot contain native pointers, slices, maps,
interfaces, channels, or functions. Readable pointer variables and parameters
do not relax that structural ownership contract. Generic bounds must prove the
field and element rules, including every allowed type argument.

Unsafe operations, reflection, and functions without Go bodies are unsupported.
Retaining interior aliases in unmarked aggregates, mutable containers, channels,
or escaping closures requires effects the checker can preserve; unsupported
transport fails explicitly. There are no suppression directives, registration
hooks, or user-declared trusted implementations. Only the fixed `im` collection
implementation supplies sealed immutable storage.

Analyze all consumer packages and tests with `immutago ./...`, or use
`go vet -vettool=<absolute path to immutago> ./...`. An ordinary Go build does not
run this proof. Unexamined dependency bodies are not certified. SDK source is
not analyzed as user code; unavailable SDK effects remain unknown for calls
that involve protected references.

The [borrowing ADR](../adr/borrowing/what.md) records the rules and verification
requirements. [Rust source attribution](../../third_party/rustc/README.md)
identifies the ported algorithms and their Go adaptations.
