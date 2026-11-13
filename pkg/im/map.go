package im

import "maps"

// FrozenMap is a read-only view of frozen map storage. Its zero value is empty.
// K and V must be immutable. The analyzer requires exclusive native ownership before
// freezing and rejects later source-handle uses. A FrozenMap copy shares storage.
type FrozenMap[K comparable, V any] struct {
	values map[K]V
}

// FreezeMap freezes values without copying its entries or allocating.
// The analyzer verifies exclusive ownership and consumes the source handle:
// aliases are rejected, and the source handle cannot be used after this call.
// FreezeMap performs no runtime validation.
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
// not change the frozen map; immutable types make deeper copying unnecessary.
func (m FrozenMap[K, V]) Clone() map[K]V { return maps.Clone(m.values) }
