package im

import "sync/atomic"

// Owned stores one pointer to an immutable LocalRef object.
// Its zero value is empty. An Owned value must not be copied.
type Owned[T any] struct {
	value atomic.Pointer[T]
}

// Load returns the current pointer, or nil if the slot is empty.
func (o *Owned[T]) Load() *T { return o.value.Load() }

// Store publishes a freshly constructed object.
func (o *Owned[T]) Store(value *T) { o.value.Store(value) }
