package immutable

import "slices"

// Slice is an immutable sequence. Its zero value is empty.
// T must be immutable, as checked by the analyzer.
type Slice[T any] struct {
	values []T
}

// NewSlice copies values into an immutable sequence.
func NewSlice[T any](values ...T) Slice[T] {
	return Slice[T]{values: slices.Clone(values)}
}

// Len returns the number of elements.
func (s Slice[T]) Len() int { return len(s.values) }

// At returns the element at index. It panics if index is out of range.
func (s Slice[T]) At(index int) T { return s.values[index] }

// Values returns an independent slice copy.
func (s Slice[T]) Values() []T { return slices.Clone(s.values) }

// Set returns a copy with one element replaced. It panics if index is out of range.
func (s Slice[T]) Set(index int, value T) Slice[T] {
	values := slices.Clone(s.values)
	values[index] = value
	return Slice[T]{values: values}
}

// Append returns a new sequence with values added at the end.
func (s Slice[T]) Append(values ...T) Slice[T] {
	result := make([]T, 0, len(s.values)+len(values))
	result = append(result, s.values...)
	result = append(result, values...)
	return Slice[T]{values: result}
}
