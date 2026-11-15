package im

import (
	"iter"
	"slices"
)

// SliceView is a local read view of slice elements. Its zero value is empty.
// The checker restricts its storage and transport. Copies share the elements.
type SliceView[E any] struct {
	values []E
}

// Slice returns a slice read view without copying elements or allocating.
func Slice[E any](value View[FrozenSlice[E]]) SliceView[E] {
	return SliceView[E]{values: value.value.values}
}

// Len returns the number of elements in O(1) time without allocations.
func (s SliceView[E]) Len() int { return len(s.values) }

// At returns a local view of the element at index without allocating.
// It copies E by value. It panics if index is out of range, as native indexing does.
func (s SliceView[E]) At(index int) View[E] { return ViewOf(s.values[index]) }

// AtPointer returns a local pointer view of an element without copying it.
// The checker requires a Frozen record E. An invalid index has Go's bounds panic.
func (s SliceView[E]) AtPointer(index int) View[*E] { return ViewOf(&s.values[index]) }

// Clone returns an independent native slice in O(n) time and space and preserves nil.
// It copies elements by value. The checker permits only Copied element types.
func (s SliceView[E]) Clone() []E { return slices.Clone(s.values) }

// All returns an iterator over index and element-view pairs in increasing index order.
// It copies each E by value and stops as soon as yield returns false.
// Use All only as the operand of a direct range loop.
func (s SliceView[E]) All() iter.Seq2[int, View[E]] {
	return func(yield func(int, View[E]) bool) {
		for index, value := range s.values {
			if !yield(index, ViewOf(value)) {
				return
			}
		}
	}
}
