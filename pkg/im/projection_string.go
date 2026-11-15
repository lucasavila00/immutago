package im

import "strings"

// StringProjection is a local read projection of string bytes. Its zero value is empty.
// The checker restricts its storage and transport. Copies share the bytes.
type StringProjection struct {
	value string
}

// ProjectString returns local read access to a native string without copying bytes or allocating.
func ProjectString[S ~string](value S) StringProjection {
	return StringProjection{value: string(value)}
}

// ProjectedString returns a string read projection without copying bytes or allocating.
func ProjectedString[S ~string](value Projection[S]) StringProjection {
	return ProjectString(value.value)
}

// Len returns the number of bytes in O(1) time without allocations.
func (s StringProjection) Len() int { return len(s.value) }

// At returns the byte at index in O(1) time without allocations.
// It panics if index is out of range, as native string indexing does.
func (s StringProjection) At(index int) byte { return s.value[index] }

// Equal compares the bytes with another local projection without copying or allocating.
func (s StringProjection) Equal(other StringProjection) bool { return s.value == other.value }

// EqualString compares the bytes with a native string without copying or allocating.
func (s StringProjection) EqualString(other string) bool { return s.value == other }

// Unsafe returns the underlying string without copying or allocating.
// The checker does not restrict this string; the caller controls its retention.
func (s StringProjection) Unsafe() string { return s.value }

// Clone returns a native string with independent storage in O(n) time and space.
// An empty result needs no allocation.
func (s StringProjection) Clone() string { return strings.Clone(s.value) }
