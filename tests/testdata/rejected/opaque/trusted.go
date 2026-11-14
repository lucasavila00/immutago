package opaque

import "github.com/lucasavila00/immutago/pkg/im"

// Table is a trusted type that other packages can name.
type Table struct{ Rows []int }

// Box holds a trusted table.
type Box struct {
	im.Shared
	Table im.Opaque[Table]
}

// Generic wraps a value whose type the package does not declare.
func Generic[T any](value *T) im.Opaque[T] { return im.Seal(value) }

// Slice wraps an unnamed type.
func Slice(rows []int) im.Opaque[[]int] { return im.Seal(&rows) }

// Rebind replaces the handle of an immutable object.
func Rebind(b *Box) { b.Table = im.Seal(&Table{}) }
