package im

import "sync/atomic"

// Cell stores a replaceable Frozen value. Its zero value has no stored value.
// Load, LoadPointer, Shared, and Store are atomic. Readers can keep an earlier view.
// The checker proves T and restricts use of the returned local views.
// A Cell must not be copied after first use.
// See https://github.com/lucasavila00/immutago/blob/main/docs/cell.md for the complete forms.
type Cell[T any] struct {
	value atomic.Pointer[T]
}

// Store publishes a private copy of value. A nonzero-sized T needs one allocation.
// It copies T by value; it does not copy the immutable storage that T refers to.
func (c *Cell[T]) Store(value T) { c.value.Store(&value) }

// Load returns a local view of the current value, or the zero T if no value was stored.
// It copies T by value without copying backing storage or allocating.
func (c *Cell[T]) Load() View[T] { return ViewOf(c.Shared()) }

// LoadPointer returns a local pointer view without copying T or allocating.
// It returns a nil pointer view before Store. The checker requires a Frozen record T.
func (c *Cell[T]) LoadPointer() View[*T] { return ViewOf(c.value.Load()) }

// Shared returns the current Frozen value for persistent use.
// It copies T by value without copying backing storage or allocating.
// If no value was stored, it returns the zero T.
func (c *Cell[T]) Shared() T {
	if value := c.value.Load(); value != nil {
		return *value
	}
	var zero T
	return zero
}
