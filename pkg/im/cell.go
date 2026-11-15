package im

import "sync/atomic"

// Cell stores a replaceable immutable snapshot. Its zero value holds the zero T.
// Load, Shared, and Store are atomic. Readers can continue to use an earlier value.
// The checker proves T and restricts use of the views returned by Load.
// A Cell must not be copied after first use.
type Cell[T any] struct {
	value atomic.Pointer[T]
}

// Store publishes a private copy of value. A nonzero-sized T needs one allocation.
// It copies T by value; it does not copy the immutable storage that T refers to.
func (c *Cell[T]) Store(value T) { c.value.Store(&value) }

// Load returns a local view of the current value, or the zero T if no value was stored.
// It copies T by value without copying backing storage or allocating.
func (c *Cell[T]) Load() View[T] { return ViewOf(c.Shared()) }

// Shared returns the current immutable snapshot for persistent use.
// It copies T by value without copying backing storage or allocating.
// If no value was stored, it returns the zero T.
func (c *Cell[T]) Shared() T {
	if value := c.value.Load(); value != nil {
		return *value
	}
	var zero T
	return zero
}
