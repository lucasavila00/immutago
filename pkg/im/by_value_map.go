package im

import (
	"iter"
	"strings"
)

// FrozenByValueMap is a local read view of map entries. Its zero value is empty.
// The checker restricts key and value types, storage, and transport. Copies share entries.
type FrozenByValueMap[K comparable, V any] struct {
	values map[K]V
}

// FreezeByValueMap makes values read-only without copying entries or allocating.
// Keep values in one local variable before Freeze. Afterward, read through the
// returned view. Run immutago to reject aliases and use of the old map.
func FreezeByValueMap[K comparable, V any](values map[K]V) FrozenByValueMap[K, V] {
	return FrozenByValueMap[K, V]{values: values}
}

// Len returns the number of entries in O(1) time without allocations.
func (m FrozenByValueMap[K, V]) Len() int { return len(m.values) }

// Get returns the value for key and whether it exists, using native map lookup.
// It allocates no storage and preserves Go's key equality rules.
func (m FrozenByValueMap[K, V]) Get(key K) (V, bool) {
	value, found := m.values[key]
	return value, found
}

// Clone returns a native map with independent storage and preserves nil.
// It copies entries by value and copies the bytes of FrozenString keys.
// Cost is O(n) plus the copied key bytes. The checker rejects Clone for local views in entries.
func (m FrozenByValueMap[K, V]) Clone() map[K]V {
	if m.values == nil {
		return nil
	}
	values := make(map[K]V, len(m.values))
	for key, value := range m.values {
		if text, ok := any(key).(FrozenString); ok {
			key = any(FrozenString{value: strings.Clone(text.value)}).(K)
		}
		values[key] = value
	}
	return values
}

// All returns an iterator over key-value pairs in unspecified order.
// It reads entries without a copy and stops as soon as yield returns false.
// Use All only as the operand of a direct range loop.
// The checker rejects All for FrozenString keys; use Get or Clone instead.
func (m FrozenByValueMap[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for key, value := range m.values {
			if !yield(key, value) {
				return
			}
		}
	}
}
