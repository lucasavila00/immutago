package im

import "sync/atomic"

// Owner stores one pointer to an immutable Scoped object.
// Its zero value is empty. An Owner value must not be copied.
type Owner[T any] struct {
	value atomic.Pointer[T]
}

// Load returns the current pointer, or nil if the slot is empty.
func (o *Owner[T]) Load() *T { return o.value.Load() }

// Store publishes a freshly constructed object.
func (o *Owner[T]) Store(value *T) { o.value.Store(value) }
