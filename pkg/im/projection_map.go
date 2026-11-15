package im

import (
	"iter"
	"maps"
)

// MapProjection is a local read projection of map entries. Its zero value is empty.
// The checker restricts its storage and transport. Copies share the entries.
type MapProjection[K comparable, V any] struct {
	values map[K]V
}

// MapOf returns a map read projection without copying entries or allocating.
func MapOf[K comparable, V any](value Projection[FrozenMap[K, V]]) MapProjection[K, V] {
	return value.value.Projection()
}

// Len returns the number of entries in O(1) time without allocations.
func (m MapProjection[K, V]) Len() int { return len(m.values) }

// Get returns a local projection of the value for key and whether it exists.
// It uses native map lookup and copies V by value without allocating.
func (m MapProjection[K, V]) Get(key K) (Projection[V], bool) {
	value, found := m.values[key]
	return ProjectionOf(value), found
}

// Clone returns an independent native map in O(n) time and space and preserves nil.
// It copies entries by value. The checker permits only Copied key and value types.
func (m MapProjection[K, V]) Clone() map[K]V { return maps.Clone(m.values) }

// All returns an iterator over key-projection and value-projection pairs in unspecified order.
// It copies each K and V by value and stops as soon as yield returns false.
// Use All only as the operand of a direct range loop.
func (m MapProjection[K, V]) All() iter.Seq2[Projection[K], Projection[V]] {
	return func(yield func(Projection[K], Projection[V]) bool) {
		for key, value := range m.values {
			if !yield(ProjectionOf(key), ProjectionOf(value)) {
				return
			}
		}
	}
}
