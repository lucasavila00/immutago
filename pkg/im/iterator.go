package im

import "iter"

// All returns an iterator over index-value pairs in increasing index order.
// It reads frozen storage directly without making a copy. Each invocation starts
// at index zero and stops as soon as yield returns false.
func (s FrozenSlice[T]) All() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		for index, value := range s.values {
			if !yield(index, value) {
				return
			}
		}
	}
}

// All returns an iterator over key-value pairs in unspecified order.
// It reads frozen storage directly without making a copy. Each invocation starts
// a new map traversal and stops as soon as yield returns false.
func (m FrozenMap[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for key, value := range m.values {
			if !yield(key, value) {
				return
			}
		}
	}
}
