package im

import "testing"

// TestUnchecked verifies the zero handle and that Value returns the trusted pointer.
func TestUnchecked(t *testing.T) {
	type table struct{ rows []int }
	var zero Unchecked[table]
	if zero.Value() != nil {
		t.Fatal("zero handle is not nil")
	}
	trusted := &table{rows: []int{1}}
	if Trust(trusted).Value() != trusted {
		t.Fatal("Value did not return the trusted pointer")
	}
}
