package api

import "github.com/lucasavila00/immutago/pkg/im"

// Table is trusted data owned by this package.
type Table struct{ Rows []int }

// Alias names the same owned type.
type Alias = Table

// Box is a named generic payload.
type Box[T any] struct{ Value T }

// New constructs a trusted table.
func New() im.Opaque[Table] { return im.Seal(&Table{Rows: []int{1}}) }

// Read uses a function value with the local type alias.
func Read(value im.Opaque[Alias]) int {
	read := im.Unseal[Alias]
	return read(value).Rows[0]
}

// Boxed wraps and reads a concrete instance declared in this package.
func Boxed() int { return im.Unseal(im.Seal(&Box[int]{Value: 1})).Value }

// Zero reads an empty handle.
func Zero() bool {
	var value im.Opaque[Table]
	return im.Unseal(value) == nil
}
