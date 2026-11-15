package im

import "strings"

// StringView is a local read view of string bytes. Its zero value is empty.
// The checker restricts its storage and transport. Copies share the bytes.
type StringView struct {
	value string
}

// Text returns a string read view without copying bytes or allocating.
func Text[S ~string](value View[S]) StringView { return StringView{value: string(value.value)} }

// Len returns the number of bytes in O(1) time without allocations.
func (s StringView) Len() int { return len(s.value) }

// At returns the byte at index in O(1) time without allocations.
// It panics if index is out of range, as native string indexing does.
func (s StringView) At(index int) byte { return s.value[index] }

// Clone returns a native string with independent storage in O(n) time and space.
// An empty result needs no allocation.
func (s StringView) Clone() string { return strings.Clone(s.value) }
