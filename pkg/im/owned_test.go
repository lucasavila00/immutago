package im

import (
	"sync"
	"testing"
)

// TestOwned verifies zero, publication, and concurrent loads.
func TestOwned(t *testing.T) {
	type model struct{ value int }
	var slot Owned[model]
	if slot.Load() != nil {
		t.Fatal("zero slot is not empty")
	}
	first := &model{value: 1}
	slot.Store(first)
	if slot.Load() != first {
		t.Fatal("load did not return the stored pointer")
	}
	var workers sync.WaitGroup
	for range 4 {
		workers.Go(func() {
			for range 100 {
				if slot.Load() == nil {
					t.Error("published slot became empty")
				}
			}
		})
	}
	for i := range 100 {
		slot.Store(&model{value: i})
	}
	workers.Wait()
	if first.value != 1 {
		t.Fatal("publication changed an earlier value")
	}
}

// BenchmarkOwnedLoad measures loading an existing pointer.
func BenchmarkOwnedLoad(b *testing.B) {
	var slot Owned[int]
	slot.Store(new(int))
	b.ReportAllocs()
	for b.Loop() {
		_ = slot.Load()
	}
}
