package im

import (
	"iter"
	"maps"
)

// MapView is a local read view of map entries. Its zero value is empty.
// The checker restricts its storage and transport. Copies share the entries.
type MapView[K comparable, V any] struct {
	values map[K]V
}

// Map returns a map read view without copying entries or allocating.
func Map[K comparable, V any](value View[FrozenMap[K, V]]) MapView[K, V] {
	return MapView[K, V]{values: value.value.values}
}

// Len returns the number of entries in O(1) time without allocations.
func (m MapView[K, V]) Len() int { return len(m.values) }

// Get returns a local view of the value for key and whether it exists.
// It uses native map lookup and copies V by value without allocating.
func (m MapView[K, V]) Get(key K) (View[V], bool) {
	value, found := m.values[key]
	return ViewOf(value), found
}

// Clone returns an independent native map in O(n) time and space and preserves nil.
// It copies entries by value. The checker permits only Copied key and value types.
func (m MapView[K, V]) Clone() map[K]V { return maps.Clone(m.values) }

// All returns an iterator over key-view and value-view pairs in unspecified order.
// It copies each K and V by value and stops as soon as yield returns false.
// Use All only as the operand of a direct range loop.
func (m MapView[K, V]) All() iter.Seq2[View[K], View[V]] {
	return func(yield func(View[K], View[V]) bool) {
		for key, value := range m.values {
			if !yield(ViewOf(key), ViewOf(value)) {
				return
			}
		}
	}
}
