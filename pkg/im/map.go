package im

import "maps"

// FrozenMap is a read-only map. Its zero value is empty.
// K and V must be immutable. Copies share the same read-only entries.
type FrozenMap[K comparable, V any] struct {
	values map[K]V
}

// FreezeMap makes values read-only without copying entries or allocating.
// Keep values in one local variable before Freeze. Afterward, read through the
// returned FrozenMap. The checker rejects aliases and use of the old map.
// Run immutago to enforce these rules; Freeze performs no runtime validation.
func FreezeMap[K comparable, V any](values map[K]V) FrozenMap[K, V] {
	return FrozenMap[K, V]{values: values}
}

// Len returns the number of entries in O(1) time without allocations.
func (m FrozenMap[K, V]) Len() int { return len(m.values) }

// Get returns the value for key and whether it exists, using native map lookup.
// It allocates no storage and preserves Go's key equality rules.
func (m FrozenMap[K, V]) Get(key K) (V, bool) { value, found := m.values[key]; return value, found }

// Clone returns an independent native map in O(n) time and space.
// It copies keys and values by value and preserves nil. Mutating the copy does
// not change the frozen map. References still point to the same immutable objects;
// immutable types make deeper copying unnecessary.
func (m FrozenMap[K, V]) Clone() map[K]V { return maps.Clone(m.values) }
