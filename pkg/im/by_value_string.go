package im

import "strings"

// FrozenByValueString is a local read view of string bytes. Its zero value is empty.
// The checker restricts storage and transport of the view. Copies share the bytes.
type FrozenByValueString struct {
	value string
}

// FreezeByValueString wraps value without copying bytes or allocating.
// The input remains usable. Run immutago to enforce the local view rules.
func FreezeByValueString(value string) FrozenByValueString { return FrozenByValueString{value: value} }

// Len returns the number of bytes in O(1) time without allocations.
func (s FrozenByValueString) Len() int { return len(s.value) }

// At returns the byte at index in O(1) time without allocations.
// It panics if index is out of range, as native string indexing does.
func (s FrozenByValueString) At(index int) byte { return s.value[index] }

// Clone returns a native string with independent storage in O(n) time and space.
// An empty result needs no allocation.
func (s FrozenByValueString) Clone() string { return strings.Clone(s.value) }
