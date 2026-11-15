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

// Equal compares the bytes with another local view without copying or allocating.
func (s StringView) Equal(other StringView) bool { return s.value == other.value }

// EqualString compares the bytes with a native string without copying or allocating.
func (s StringView) EqualString(other string) bool { return s.value == other }

// Unsafe returns the underlying string without copying or allocating.
// The checker does not restrict this string; the caller controls its retention.
func (s StringView) Unsafe() string { return s.value }

// Clone returns a native string with independent storage in O(n) time and space.
// An empty result needs no allocation.
func (s StringView) Clone() string { return strings.Clone(s.value) }
