// Package flow contains the graph engines used by the borrow checker.
// Backward liveness is adapted
// from rustc commit 80196be0e787008b6fb14d2990aba92083cd68e1.
// See third_party/rustc for source attribution and the MIT license.
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
