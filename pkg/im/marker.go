// Package im provides a marker and immutable value collections.
//
// Embed Marker in a struct to require immutable fields. Run the immutago
// analyzer on all consumer packages; Go's type checker alone does not enforce
// the field or collection element rules. Construct marked values with literals.
// Variables can be replaced, but fields of marked values must not be changed.
//
// Slice and Map copy their inputs and outputs. Updates return new collections.
// Their zero values are empty collections. Element and key types must themselves
// be immutable; the analyzer checks this requirement, including nested types.
package im

// Marker opts a struct into immutability checks when embedded by value.
// A named field or pointer embedding does not satisfy the marker contract.
type Marker struct{}
