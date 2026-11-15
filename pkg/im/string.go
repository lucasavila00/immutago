package im

import "strings"

// FrozenString holds read-only bytes. Its zero value is empty.
// Copies share the same bytes. Native string access requires Clone.
type FrozenString struct {
	value string
}

// FreezeString copies value into independent storage in O(n) time and space.
// It does not retain a larger string allocation when value is a substring.
// The input remains usable. An empty input needs no allocation.
func FreezeString(value string) FrozenString { return FrozenString{value: strings.Clone(value)} }

// Len returns the number of bytes in O(1) time without allocations.
func (s FrozenString) Len() int { return len(s.value) }

// At returns the byte at index in O(1) time without allocations.
// It panics if index is out of range, as native string indexing does.
func (s FrozenString) At(index int) byte { return s.value[index] }

// Clone returns a native string with independent storage in O(n) time and space.
// Substrings of the result do not retain this FrozenString's storage.
// An empty result needs no allocation.
func (s FrozenString) Clone() string { return strings.Clone(s.value) }
