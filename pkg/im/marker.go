// Package im provides a marker and immutable value collections.
//
// Embed Marker in a struct to require immutable fields. Run the immutago
// analyzer on all consumer packages; Go's type checker alone does not enforce
// the field or collection element rules. Owned local values permit construction
// and updates. Shared references permit reads; writes to borrowed storage fail
// while a conflicting loan is live. A value copy has independent storage.
//
// FreezeSlice and FreezeMap share their input storage without copying. The analyzer
// rejects later writes through native aliases. Clone returns an independent copy.
// Their zero values are empty collections. Element and key types must themselves
// be immutable; the analyzer checks this requirement, including nested types.
package im

// Marker opts a struct into immutability checks when embedded by value.
// A named field or pointer embedding does not satisfy the marker contract.
type Marker struct{}
