package im

import "hash/maphash"

// Map is an immutable mapping. Its zero value is empty.
// K and V must be immutable, as checked by the analyzer. A persistent hash trie
// shares unchanged branches between versions. Reads do not allocate.
type Map[K comparable, V any] struct {
	root   *mapNode[K, V]
	seed   maphash.Seed
	length int
}

// NewMap copies values into an immutable mapping.
func NewMap[K comparable, V any](values map[K]V) Map[K, V] {
	var result Map[K, V]
	for key, value := range values {
		result = result.Set(key, value)
	}
	return result
}

// Len returns the number of entries in O(1) time without allocations.
func (m Map[K, V]) Len() int { return m.length }

// Get returns the value for key and whether the key exists without allocations.
// Lookup visits at most 13 trie levels plus any full-hash collision bucket.
func (m Map[K, V]) Get(key K) (V, bool) {
	if m.root != nil {
		return getMapNode(m.root, maphash.Comparable(m.seed, key), key)
	}
	var zero V
	return zero, false
}

// Values returns an independent map copy in O(n) time.
func (m Map[K, V]) Values() map[K]V {
	values := make(map[K]V, m.length)
	copyMapValues(m.root, values)
	return values
}

// Set returns a map with key set to value, sharing all unchanged branches.
// It copies at most 13 trie levels plus any full-hash collision bucket.
func (m Map[K, V]) Set(key K, value V) Map[K, V] {
	if m.root == nil {
		m.seed = maphash.MakeSeed()
	}
	var added bool
	m.root, added = setMapNode(m.root, 0, maphash.Comparable(m.seed, key), key, value)
	if added {
		m.length++
	}
	return m
}

// Delete returns a map without key, sharing all unchanged branches.
// It copies at most 13 trie levels plus any full-hash collision bucket.
func (m Map[K, V]) Delete(key K) Map[K, V] {
	if m.root == nil {
		return m
	}
	var removed bool
	m.root, removed = deleteMapNode(m.root, 0, maphash.Comparable(m.seed, key), key)
	if removed {
		m.length--
	}
	return m
}
