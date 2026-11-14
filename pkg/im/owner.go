package im

import "sync/atomic"

// Owner stores one pointer to an immutable Scoped object.
// Declare a slot as a package variable or a direct named field of an unmarked struct.
// Its zero value is empty. An Owner value must not be copied.
// See https://github.com/lucasavila00/immutago/blob/main/docs/owner.md for the complete forms.
type Owner[T any] struct {
	value atomic.Pointer[T]
}

// Load returns the current pointer, or nil if the slot is empty.
// Keep the pointer local or pass it to a function that does not save it.
// It remains valid after a later Store.
// Go's garbage collector manages its lifetime.
func (o *Owner[T]) Load() *T { return o.value.Load() }

// Store publishes a freshly constructed object.
func (o *Owner[T]) Store(value *T) { o.value.Store(value) }
