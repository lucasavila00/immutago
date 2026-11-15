package im

import (
	"iter"
	"slices"
)

// SliceProjection is a local read projection of slice elements. Its zero value is empty.
// The checker restricts its storage and transport. Copies share the elements.
type SliceProjection[E any] struct {
	values []E
}

// Slice returns a slice read projection without copying elements or allocating.
func Slice[E any](value Projection[FrozenSlice[E]]) SliceProjection[E] {
	return SliceProjection[E]{values: value.value.values}
}

// Len returns the number of elements in O(1) time without allocations.
func (s SliceProjection[E]) Len() int { return len(s.values) }

// At returns a local projection of the element at index without allocating.
// It copies E by value. It panics if index is out of range, as native indexing does.
func (s SliceProjection[E]) At(index int) Projection[E] { return ProjectionOf(s.values[index]) }

// AtPointer returns a local pointer projection of an element without copying it.
// The checker requires a Frozen record E. An invalid index has Go's bounds panic.
func (s SliceProjection[E]) AtPointer(index int) Projection[*E] {
	return ProjectionOf(&s.values[index])
}

// Clone returns an independent native slice in O(n) time and space and preserves nil.
// It copies elements by value. The checker permits only Copied element types.
func (s SliceProjection[E]) Clone() []E { return slices.Clone(s.values) }

// All returns an iterator over index and element-projection pairs in increasing index order.
// It copies each E by value and stops as soon as yield returns false.
// Use All only as the operand of a direct range loop.
func (s SliceProjection[E]) All() iter.Seq2[int, Projection[E]] {
	return func(yield func(int, Projection[E]) bool) {
		for index, value := range s.values {
			if !yield(index, ProjectionOf(value)) {
				return
			}
		}
	}
}
