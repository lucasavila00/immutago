package immutable

import (
	"math/bits"
	"slices"
)

const mapBits = 5
const mapMask = (1 << mapBits) - 1

// mapEntry stores a value and its equality key in a full-hash collision bucket.
type mapEntry[K comparable, V any] struct {
	key   K
	value V
}

// mapNode holds a leaf bucket or bitmap-indexed children for five hash bits.
type mapNode[K comparable, V any] struct {
	hash     uint64
	entries  []mapEntry[K, V]
	bitmap   uint32
	children []*mapNode[K, V]
}

// getMapNode follows hash bits and compares keys only in the matching leaf.
func getMapNode[K comparable, V any](node *mapNode[K, V], hash uint64, key K) (V, bool) {
	for shift := uint(0); node != nil; shift += mapBits {
		if node.entries != nil {
			if node.hash == hash {
				for _, entry := range node.entries {
					if entry.key == key {
						return entry.value, true
					}
				}
			}
			break
		}
		bit := uint32(1) << ((hash >> shift) & mapMask)
		if node.bitmap&bit == 0 {
			break
		}
		node = node.children[bits.OnesCount32(node.bitmap&(bit-1))]
	}
	var zero V
	return zero, false
}

// setMapNode copies an insertion path and returns whether a new key was added.
func setMapNode[K comparable, V any](
	node *mapNode[K, V], shift uint, hash uint64, key K, value V,
) (*mapNode[K, V], bool) {
	if node == nil {
		return mapLeaf(hash, key, value), true
	}
	if node.entries != nil {
		if node.hash != hash {
			return mergeMapLeaves(node, mapLeaf(hash, key, value), shift), true
		}
		return setMapEntry(node, key, value)
	}
	bit := uint32(1) << ((hash >> shift) & mapMask)
	index := bits.OnesCount32(node.bitmap & (bit - 1))
	if node.bitmap&bit == 0 {
		children := make([]*mapNode[K, V], len(node.children)+1)
		copy(children, node.children[:index])
		children[index] = mapLeaf(hash, key, value)
		copy(children[index+1:], node.children[index:])
		return &mapNode[K, V]{bitmap: node.bitmap | bit, children: children}, true
	}
	child, added := setMapNode(node.children[index], shift+mapBits, hash, key, value)
	children := slices.Clone(node.children)
	children[index] = child
	return &mapNode[K, V]{bitmap: node.bitmap, children: children}, added
}

// mapLeaf creates a one-entry leaf without any child storage.
func mapLeaf[K comparable, V any](hash uint64, key K, value V) *mapNode[K, V] {
	return &mapNode[K, V]{hash: hash, entries: []mapEntry[K, V]{{key: key, value: value}}}
}

// setMapEntry copies one collision bucket and preserves Go key equality semantics.
func setMapEntry[K comparable, V any](node *mapNode[K, V], key K, value V) (*mapNode[K, V], bool) {
	for index, entry := range node.entries {
		if entry.key == key {
			entries := slices.Clone(node.entries)
			entries[index] = mapEntry[K, V]{key: key, value: value}
			return &mapNode[K, V]{hash: node.hash, entries: entries}, false
		}
	}
	entries := make([]mapEntry[K, V], len(node.entries)+1)
	copy(entries, node.entries)
	entries[len(node.entries)] = mapEntry[K, V]{key: key, value: value}
	return &mapNode[K, V]{hash: node.hash, entries: entries}, true
}

// mergeMapLeaves constructs the shortest branch path that separates two distinct hashes.
func mergeMapLeaves[K comparable, V any](left, right *mapNode[K, V], shift uint) *mapNode[K, V] {
	leftBit := uint32(1) << ((left.hash >> shift) & mapMask)
	rightBit := uint32(1) << ((right.hash >> shift) & mapMask)
	if leftBit == rightBit {
		child := mergeMapLeaves(left, right, shift+mapBits)
		return &mapNode[K, V]{bitmap: leftBit, children: []*mapNode[K, V]{child}}
	}
	children := []*mapNode[K, V]{left, right}
	if leftBit > rightBit {
		children[0], children[1] = right, left
	}
	return &mapNode[K, V]{bitmap: leftBit | rightBit, children: children}
}

// deleteMapNode copies a removal path, retaining the original node for a missing key.
func deleteMapNode[K comparable, V any](node *mapNode[K, V], shift uint, hash uint64, key K) (*mapNode[K, V], bool) {
	if node.entries != nil {
		return deleteMapEntry(node, hash, key)
	}
	bit := uint32(1) << ((hash >> shift) & mapMask)
	if node.bitmap&bit == 0 {
		return node, false
	}
	index := bits.OnesCount32(node.bitmap & (bit - 1))
	child, removed := deleteMapNode(node.children[index], shift+mapBits, hash, key)
	if !removed {
		return node, false
	}
	bitmap := node.bitmap
	var children []*mapNode[K, V]
	if child == nil {
		bitmap &^= bit
		children = make([]*mapNode[K, V], len(node.children)-1)
		copy(children, node.children[:index])
		copy(children[index:], node.children[index+1:])
	} else {
		children = slices.Clone(node.children)
		children[index] = child
	}
	if len(children) == 0 {
		return nil, true
	}
	if len(children) == 1 && children[0].entries != nil {
		return children[0], true
	}
	return &mapNode[K, V]{bitmap: bitmap, children: children}, true
}

// deleteMapEntry removes an equal key from a matching collision bucket.
func deleteMapEntry[K comparable, V any](node *mapNode[K, V], hash uint64, key K) (*mapNode[K, V], bool) {
	if node.hash != hash {
		return node, false
	}
	for index, entry := range node.entries {
		if entry.key != key {
			continue
		}
		if len(node.entries) == 1 {
			return nil, true
		}
		entries := make([]mapEntry[K, V], len(node.entries)-1)
		copy(entries, node.entries[:index])
		copy(entries[index:], node.entries[index+1:])
		return &mapNode[K, V]{hash: hash, entries: entries}, true
	}
	return node, false
}

// copyMapValues visits each leaf once and copies entries into caller-owned storage.
func copyMapValues[K comparable, V any](node *mapNode[K, V], values map[K]V) {
	if node == nil {
		return
	}
	for _, entry := range node.entries {
		values[entry.key] = entry.value
	}
	for _, child := range node.children {
		copyMapValues(child, values)
	}
}
