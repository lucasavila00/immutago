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

// TestCounterGettersKeepTarget checks generated value and pointer getters.
func TestCounterGettersKeepTarget(t *testing.T) {
	value := api.NewCounter()
	pointer := new(value)
	var cell CounterSnapshotCell
	cell.Store(CounterSnapshot{Counter: value, Pointer: pointer})
	projection := cell.Load()
	copied, retained := ReadCounterSnapshot(projection)
	if retained != pointer || copied.Add(1) != 1 || retained.Add(1) != 2 {
		t.Fatal("generated getters changed the handle target")
	}
}
