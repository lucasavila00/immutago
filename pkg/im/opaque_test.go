package im

import "testing"

// TestOpaque verifies the zero handle and that Unseal returns the trusted pointer.
func TestOpaque(t *testing.T) {
	type table struct{ rows []int }
	var zero Opaque[table]
	if Unseal(zero) != nil {
		t.Fatal("zero handle is not nil")
	}
	trusted := &table{rows: []int{1}}
	if Unseal(Seal(trusted)) != trusted {
		t.Fatal("Unseal did not return the trusted pointer")
	}
}

// TestOpaqueHasNoReaderMethod prevents interface calls and promoted methods from exposing the pointer.
func TestOpaqueHasNoReaderMethod(t *testing.T) {
	type table struct{ rows []int }
	type wrapper struct{ Opaque[table] }
	for _, value := range []any{Opaque[table]{}, new(Opaque[table]), wrapper{}, new(wrapper)} {
		if _, ok := value.(interface{ Value() *table }); ok {
			t.Fatal("opaque handle exposes a pointer through an interface method")
		}
	}
}
