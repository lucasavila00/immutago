package consumer

import "testing"

func TestGenericReads(t *testing.T) {
	if got := GenericReads(); got != 12 {
		t.Fatalf("GenericReads() = %d; want 12", got)
	}
}
