package valid

import (
	"example.com/consumer/api"
	"testing"
)

// TestCounterCopiesShareTarget checks the mutable payload without copying it.
func TestCounterCopiesShareTarget(t *testing.T) {
	first := api.NewCounter()
	other := first
	next := KeepCounter(other)
	if first.Add(2) != 2 || next() != 3 {
		t.Fatal("counter copies did not share one target")
	}
}
