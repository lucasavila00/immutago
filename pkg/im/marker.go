// Package im provides markers and immutable value collections.
//
// Embed Shared or Scoped in a struct to require immutable fields. Run the
// immutago analyzer on all consumer packages; Go's type checker alone does not
// enforce the field or collection element rules. Both markers make objects
// immutable from construction. Whole variables can be rebound to another value.
//
// Pointers to Shared objects can be copied, stored, and returned. Pointers to
// Scoped objects are restricted to local use and direct typed calls that do
// not let the pointer escape. Go manages object lifetimes. A value copy
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

// Scoped marks an immutable struct whose pointers must not escape.
// Embed it by value. A named field or pointer embedding does not mark the struct.
type Scoped struct{}

// Process marks a mutable struct that runs as a supervised process.
// Only proc.Start creates it, and other code calls its exported methods through a proc.Ref.
// Embed it by value. A named field or pointer embedding does not mark the struct.
type Process struct{}
