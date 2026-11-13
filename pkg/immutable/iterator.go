package immutable

import "iter"

// All returns an iterator over index-value pairs in increasing index order.
// It visits shared storage directly in O(n) time without making a collection copy.
// The iterator can be reused and retains the value on which All was called.
func (s Slice[T]) All() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		index := 0
		if !yieldSliceNode(s.root, &index, yield) {
			return
		}
		for _, value := range s.tail {
			if !yield(index, value) {
				return
			}
			index++
		}
	}
}

// All returns an iterator over key-value pairs in unspecified order.
// It visits shared storage directly in O(n) time without making a collection copy.
// The iterator can be reused and retains the value on which All was called.
func (m Map[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		yieldMapNode(m.root, yield)
	}
}

// yieldSliceNode visits leaves in order and stops immediately when yield returns false.
func yieldSliceNode[T any](node *sliceNode[T], index *int, yield func(int, T) bool) bool {
	if node == nil {
		return true
	}
	for _, value := range node.values {
		if !yield(*index, value) {
			return false
		}
		*index++
	}
	for _, child := range node.children {
		if !yieldSliceNode(child, index, yield) {
			return false
		}
	}
	return true
}

// yieldMapNode visits each entry once and propagates early termination through all branches.
func yieldMapNode[K comparable, V any](node *mapNode[K, V], yield func(K, V) bool) bool {
	if node == nil {
		return true
	}
	for _, entry := range node.entries {
		if !yield(entry.key, entry.value) {
			return false
		}
	}
	for _, child := range node.children {
		if !yieldMapNode(child, yield) {
			return false
		}
	}
	return true
}
