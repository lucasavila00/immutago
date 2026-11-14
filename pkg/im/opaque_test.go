package im

import "testing"

// TestOpaque verifies the zero handle and that Value returns the trusted pointer.
func TestOpaque(t *testing.T) {
	type table struct{ rows []int }
	var zero Opaque[table]
	if zero.Value() != nil {
		t.Fatal("zero handle is not nil")
	}
	trusted := &table{rows: []int{1}}
	if Seal(trusted).Value() != trusted {
		t.Fatal("Value did not return the trusted pointer")
	}
}
