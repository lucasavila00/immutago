package im

import "slices"

// FrozenSlice is a read-only view of frozen slice storage. Its zero value is empty.
// T must be immutable. The analyzer requires exclusive native ownership before
// freezing and rejects later source-handle uses. A FrozenSlice copy shares storage.
type FrozenSlice[T any] struct {
	values []T
}

// FreezeSlice freezes values without copying its backing array or allocating.
// The analyzer verifies exclusive ownership and consumes the source handle:
// aliases are rejected, and the source handle cannot be used after this call.
// FreezeSlice performs no runtime validation.
func FreezeSlice[T any](values []T) FrozenSlice[T] { return FrozenSlice[T]{values: values} }

// Len returns the number of elements in O(1) time without allocations.
func (s FrozenSlice[T]) Len() int { return len(s.values) }

// At returns the element at index in O(1) time without allocations.
// It panics if index is out of range, as native slice indexing does.
func (s FrozenSlice[T]) At(index int) T { return s.values[index] }

// Clone returns an independent native slice in O(n) time and space.
// It copies elements by value and preserves nil. Mutating the copy does not
// change the frozen slice; immutable element types make deeper copying unnecessary.
func (s FrozenSlice[T]) Clone() []T { return slices.Clone(s.values) }
