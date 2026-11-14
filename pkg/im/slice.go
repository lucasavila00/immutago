package im

import "slices"

// FrozenSlice is a read-only slice. Its zero value is empty.
// T must be immutable. Copies share the same read-only entries.
type FrozenSlice[T any] struct {
	values []T
}

// FreezeSlice makes values read-only without copying entries or allocating.
// Keep values in one local variable before Freeze. Afterward, read through the
// returned FrozenSlice. The checker rejects aliases and use of the old slice.
// Run immutago to enforce these rules; Freeze performs no runtime validation.
func FreezeSlice[T any](values []T) FrozenSlice[T] { return FrozenSlice[T]{values: values} }

// Len returns the number of elements in O(1) time without allocations.
func (s FrozenSlice[T]) Len() int { return len(s.values) }

// At returns the element at index in O(1) time without allocations.
// It panics if index is out of range, as native slice indexing does.
func (s FrozenSlice[T]) At(index int) T { return s.values[index] }

// Clone returns an independent native slice in O(n) time and space.
// It copies elements by value and preserves nil. Mutating the copy does not
// change the frozen slice. References still point to the same immutable objects;
// immutable element types make deeper copying unnecessary.
func (s FrozenSlice[T]) Clone() []T { return slices.Clone(s.values) }
