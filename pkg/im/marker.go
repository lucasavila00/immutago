// Package im provides a marker and immutable value collections.
//
// Embed Marker in a struct to require immutable fields. Run the immutago
// analyzer on all consumer packages; Go's type checker alone does not enforce
// the field or collection element rules. Marked objects are immutable from
// construction. Shared references can be copied, stored, and returned. Go manages
// their lifetime. A value copy preserves references to the same immutable objects.
//
// FreezeSlice and FreezeMap share their input storage without copying. The analyzer
// requires exclusive ownership and rejects later source-handle uses. Clone returns
// an independent copy.
// Their zero values are empty collections. Element and key types must themselves
// be immutable; the analyzer checks this requirement, including nested types.
package im

// Marker opts a struct into immutability checks when embedded by value.
// A named field or pointer embedding does not satisfy the marker contract.
type Marker struct{}
