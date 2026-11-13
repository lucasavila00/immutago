// Package flow supplies graph and set operations for builder consumption checks.
package flow

// Set stores integer identities. Present entries always have value true.
type Set map[int]bool

// NewSet returns a set containing values.
func NewSet(values ...int) Set {
	result := make(Set, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

// Add inserts value and reports whether the set changed.
func (s Set) Add(value int) bool {
	if s[value] {
		return false
	}
	s[value] = true
	return true
}
