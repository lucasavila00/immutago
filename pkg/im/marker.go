// Package im provides markers and immutable value collections.
//
// Embed Immut or ImmutNoRef in a struct to require immutable fields. Run the
// immutago analyzer on all consumer packages; Go's type checker alone does not
// enforce the field or collection element rules. Both markers make objects
// immutable from construction. Whole variables can be rebound to another value.
//
// References to Immut objects can be copied, stored, and returned. References to
// ImmutNoRef objects are restricted to local use and direct typed calls that do
// not let the reference escape. Go manages object lifetimes. A value copy
// preserves references to the same immutable objects.
//
// FreezeSlice and FreezeMap share their input storage without copying. The analyzer
// requires exclusive ownership and rejects later source-handle uses. Clone returns
// an independent copy.
// Their zero values are empty collections. Element and key types must themselves
// be immutable; the analyzer checks this requirement, including nested types.
package im

// Immut marks an immutable struct whose shared references may be stored or returned.
// Embed it by value. A named field or pointer embedding does not mark the struct.
type Immut struct{}

// ImmutNoRef marks an immutable struct whose references must not escape.
// Embed it by value. A named field or pointer embedding does not mark the struct.
type ImmutNoRef struct{}
