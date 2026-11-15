package im

import "strings"

// FrozenString is a read-only string. Its zero value is empty.
// Copies share the same immutable bytes. It has no substring or raw-string getter.
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

// Clone returns a native string with independent storage in O(n) time and space.
// An empty result needs no allocation.
func (s FrozenString) Clone() string { return strings.Clone(s.value) }

// String returns a local string view without copying bytes or allocating.
func String(value View[FrozenString]) StringView { return StringView{value: value.value.value} }
