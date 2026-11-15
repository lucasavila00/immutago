package im

import "strings"

// FrozenString holds read-only bytes. Its zero value is empty.
// Copies share the source bytes. Native string access requires Clone.
type FrozenString struct {
	value string
}

// FreezeString wraps value without copying bytes or allocating.
// The input remains usable because native string bytes are immutable.
func FreezeString(value string) FrozenString { return FrozenString{value: value} }

// ByValue returns a local read view without copying bytes or allocating.
// Run immutago to enforce the view's storage and transport rules.
func (s FrozenString) ByValue() FrozenByValueString { return FrozenByValueString{value: s.value} }

// Len returns the number of bytes in O(1) time without allocations.
func (s FrozenString) Len() int { return len(s.value) }

// At returns the byte at index in O(1) time without allocations.
// It panics if index is out of range, as native string indexing does.
func (s FrozenString) At(index int) byte { return s.value[index] }

// Clone returns a native string with independent storage in O(n) time and space.
// An empty result needs no allocation.
func (s FrozenString) Clone() string { return strings.Clone(s.value) }
