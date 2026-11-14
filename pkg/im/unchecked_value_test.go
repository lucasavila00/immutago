package im

import "testing"

// TestUncheckedValue verifies the zero handle and that Unwrap returns the trusted pointer.
func TestUncheckedValue(t *testing.T) {
	type table struct{ rows []int }
	var zero UncheckedValue[table]
	if Unwrap(zero) != nil {
		t.Fatal("zero handle is not nil")
	}
	trusted := &table{rows: []int{1}}
	if Unwrap(Trust(trusted)) != trusted {
		t.Fatal("Unwrap did not return the trusted pointer")
	}
}

// TestUncheckedValueHasNoReaderMethod prevents interface calls and promoted methods from exposing the pointer.
func TestUncheckedValueHasNoReaderMethod(t *testing.T) {
	type table struct{}
	type wrapper struct{ UncheckedValue[table] }
	for _, value := range []any{UncheckedValue[table]{}, new(UncheckedValue[table]), wrapper{}, new(wrapper)} {
		if _, ok := value.(interface{ Value() *table }); ok {
			t.Fatal("unchecked value exposes a pointer through an interface method")
		}
	}
}
