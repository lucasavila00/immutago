package accepted

import "testing"

func TestGenericReads(t *testing.T) {
	if got := GenericReads(); got != 27 {
		t.Fatalf("GenericReads() = %d; want 27", got)
	}
}
