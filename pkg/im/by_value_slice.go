package im

import (
	"iter"
	"slices"
)

// FrozenByValueSlice is a local read view of slice entries. Its zero value is empty.
// The checker restricts element types, storage, and transport. Copies share entries.
type FrozenByValueSlice[T any] struct {
	values []T
}

// FreezeByValueSlice makes values read-only without copying entries or allocating.
// Keep values in one local variable before Freeze. Afterward, read through the
// returned view. Run immutago to reject aliases and use of the old slice.
func FreezeByValueSlice[T any](values []T) FrozenByValueSlice[T] {
	return FrozenByValueSlice[T]{values: values}
}

// Len returns the number of elements in O(1) time without allocations.
func (s FrozenByValueSlice[T]) Len() int { return len(s.values) }

// At returns the element at index in O(1) time without allocations.
// It panics if index is out of range, as native slice indexing does.
func (s FrozenByValueSlice[T]) At(index int) T { return s.values[index] }

// Clone returns an independent native slice in O(n) time and space and preserves nil.
// It copies elements by value. The checker rejects Clone for elements with local views.
func (s FrozenByValueSlice[T]) Clone() []T { return slices.Clone(s.values) }

// All returns an iterator over index-value pairs in increasing index order.
// It reads entries without a copy and stops as soon as yield returns false.
// Use All only as the operand of a direct range loop.
func (s FrozenByValueSlice[T]) All() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		for index, value := range s.values {
			if !yield(index, value) {
				return
			}
		}
	}
}
