package immutable

import "maps"

// Map is an immutable mapping. Its zero value is empty.
// K and V must be immutable, as checked by the analyzer.
type Map[K comparable, V any] struct {
	values map[K]V
}

// NewMap copies values into an immutable mapping.
func NewMap[K comparable, V any](values map[K]V) Map[K, V] {
	return Map[K, V]{values: maps.Clone(values)}
}

// Len returns the number of entries.
func (m Map[K, V]) Len() int { return len(m.values) }

// Get returns the value for key and whether the key exists.
func (m Map[K, V]) Get(key K) (V, bool) {
	value, ok := m.values[key]
	return value, ok
}

// Values returns an independent map copy.
func (m Map[K, V]) Values() map[K]V { return maps.Clone(m.values) }

// Set returns a copy with key set to value.
func (m Map[K, V]) Set(key K, value V) Map[K, V] {
	values := make(map[K]V, len(m.values)+1)
	maps.Copy(values, m.values)
	values[key] = value
	return Map[K, V]{values: values}
}

// Delete returns a copy without key.
func (m Map[K, V]) Delete(key K) Map[K, V] {
	values := maps.Clone(m.values)
	delete(values, key)
	return Map[K, V]{values: values}
}
