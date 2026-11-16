package im

import "strings"

// FrozenString is a read-only string. Its zero value is empty.
// Copies share the same immutable bytes.
type FrozenString struct {
	value string
}

// FreezeString wraps value without copying bytes or allocating.
// Go strings are immutable, so value remains usable. A substring input still
// shares its original backing allocation; use Clone for independent storage.
func FreezeString(value string) FrozenString { return FrozenString{value: value} }

// Len returns the number of bytes without allocating.
func (s FrozenString) Len() int { return len(s.value) }

// At returns the byte at index without allocating.
// It panics if index is out of range, as native string indexing does.
func (s FrozenString) At(index int) byte { return s.value[index] }

// Equal compares the bytes with another frozen string without copying or allocating.
func (s FrozenString) Equal(other FrozenString) bool { return s.value == other.value }

// EqualString compares the bytes with a native string without copying or allocating.
func (s FrozenString) EqualString(other string) bool { return s.value == other }

// Unchecked returns the underlying string without copying or allocating.
// The caller controls retention of its shared backing storage.
func (s FrozenString) Unchecked() string { return s.value }

// Clone returns a native string with independent storage in O(n) time and space.
// An empty result needs no allocation.
func (s FrozenString) Clone() string { return strings.Clone(s.value) }

// StringOfFrozen returns a local string projection without copying bytes or allocating.
func StringOfFrozen(value Projection[FrozenString]) StringProjection {
	return value.value.Projection()
}

// Projection returns local read access without copying backing storage or allocating.
func (s FrozenString) Projection() StringProjection { return StringProjection{value: s.value} }
