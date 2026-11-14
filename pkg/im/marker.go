// Package im provides markers and immutable value collections.
//
// Embed Shared or Scoped in a struct to require immutable fields. Run the
// immutago analyzer on all consumer packages; Go's type checker alone does not
// enforce the field or collection element rules. Both markers make objects
// immutable from construction. Whole variables can be rebound to another value.
//
// Pointers to Shared objects can be copied, stored, and returned. Pointers to
// Scoped objects stay in local bindings or direct typed calls that do not retain
// them. Owner publishes fresh Scoped objects; its loaded pointers follow the
// same local-use rules. These mutation and retention restrictions add to Go's
// memory safety. Go's garbage collector manages object lifetimes. A value copy
// preserves pointers to the same immutable objects.
//
// FreezeSlice and FreezeMap share their input storage without copying. The analyzer
// requires exclusive ownership and rejects later source-handle uses. Clone returns
// an independent copy.
// Their zero values are empty collections. Element and key types must themselves
// be immutable; the analyzer checks this requirement, including nested types.
package im

// Shared marks an immutable struct whose shared pointers may be stored or returned.
// Embed it by value. A named field or pointer embedding does not mark the struct.
type Shared struct{}

// Scoped marks an immutable struct whose pointers have restricted retention.
// Owner is the supported publication slot for fresh Scoped objects.
// Embed it by value. A named field or pointer embedding does not mark the struct.
type Scoped struct{}
