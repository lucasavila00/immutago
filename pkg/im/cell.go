package im

import "sync/atomic"

// Cell stores a replaceable Frozen value. Its zero value has no stored value.
// Load, LoadPointer, LoadShared, and Store are atomic. Readers can keep an earlier projection.
// The checker proves T and restricts use of the returned local projections.
// A Cell must not be copied after first use.
// See https://github.com/lucasavila00/immutago/blob/main/docs/reference/cell.md for the complete forms.
type Cell[T any] struct {
	value atomic.Pointer[T]
}

// Store publishes a private copy of value. A nonzero-sized T needs one allocation.
// It copies T by value; it does not copy the immutable storage that T refers to.
func (c *Cell[T]) Store(value T) { c.value.Store(&value) }

// Load returns a local projection of the current value, or the zero T if no value was stored.
// It copies T by value without copying backing storage or allocating.
func (c *Cell[T]) Load() Projection[T] { return ProjectionOf(c.LoadShared()) }

// LoadPointer returns a local pointer projection without copying T or allocating.
// It returns a nil pointer projection before Store. The checker requires a Frozen record T.
func (c *Cell[T]) LoadPointer() Projection[*T] { return ProjectionOf(c.value.Load()) }

// LoadShared returns the current Frozen value for persistent use.
// It copies T by value without copying backing storage or allocating.
// If no value was stored, it returns the zero T.
func (c *Cell[T]) LoadShared() T {
	if value := c.value.Load(); value != nil {
		return *value
	}
	var zero T
	return zero
}
