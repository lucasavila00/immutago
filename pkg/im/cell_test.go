package im

import (
	"sync"
	"testing"
)

type testSnapshot struct {
	Snapshot
	Number int
	Text   string
	Slice  FrozenSlice[int]
	Map    FrozenMap[int, int]
	Child  *testChild
	Array  [2]int
}

type testChild struct {
	Snapshot
	Number int
}

// TestCellZero checks the zero snapshot, scalar view, and nil pointer view.
func TestCellZero(t *testing.T) {
	var scalar Cell[int]
	if Copy(scalar.Load()) != 0 || scalar.Shared() != 0 {
		t.Fatal("zero scalar cell changed the zero value")
	}
	var pointer Cell[*testChild]
	if !IsNil(pointer.Load()) || pointer.Shared() != nil {
		t.Fatal("zero pointer cell is not nil")
	}
	var record Cell[testSnapshot]
	if Copy(Select(record.Load(), func(v testSnapshot) int { return v.Number })) != 0 {
		t.Fatal("zero record cell changed the zero value")
	}
}

// TestCellStore checks the private root copy and shared immutable backing storage.
func TestCellStore(t *testing.T) {
	input := testSnapshot{Number: 1, Slice: FreezeSlice([]int{2}), Child: &testChild{Number: 3}}
	var cell Cell[testSnapshot]
	cell.Store(input)
	if cell.value.Load() == &input {
		t.Fatal("cell kept the caller's root address")
	}
	input.Number = 9
	first := cell.Load()
	shared := cell.Shared()
	if first.value.Number != 1 || shared.Number != 1 {
		t.Fatal("cell root changed with the caller's value")
	}
	if &shared.Slice.values[0] != &input.Slice.values[0] || shared.Child != input.Child {
		t.Fatal("cell copied immutable backing storage")
	}
	shared.Number = 8
	if cell.Shared().Number != 1 {
		t.Fatal("Shared returned the cell's root storage")
	}
	cell.Store(testSnapshot{Number: 4})
	if first.value.Number != 1 || cell.Shared().Number != 4 {
		t.Fatal("Store changed an earlier snapshot")
	}
}

// TestCellConcurrent checks concurrent publication of complete root values.
func TestCellConcurrent(t *testing.T) {
	var cell Cell[[2]int]
	var workers sync.WaitGroup
	for range 4 {
		workers.Go(func() {
			for range 100 {
				value := Copy(cell.Load())
				if value[0] != value[1] {
					t.Error("load mixed two stored values")
				}
			}
		})
	}
	for i := range 100 {
		cell.Store([2]int{i, i})
	}
	workers.Wait()
}
