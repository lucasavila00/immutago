package im

import (
	"sync"
	"testing"
)

type testFrozen struct {
	Frozen
	Number int
	Text   string
	Slice  FrozenSlice[int]
	Map    FrozenMap[int, int]
	Child  *testChild
	Array  [2]int
}

type testChild struct {
	Frozen
	Number int
}

// TestCellZero checks the zero snapshot, scalar projection, and nil pointer projection.
func TestCellZero(t *testing.T) {
	var scalar Cell[int]
	if Value(scalar.Load()) != 0 || scalar.LoadShared() != 0 {
		t.Fatal("zero scalar cell changed the zero value")
	}
	var pointer Cell[*testChild]
	if !IsNil(pointer.Load()) || pointer.LoadShared() != nil {
		t.Fatal("zero pointer cell is not nil")
	}
	var record Cell[testFrozen]
	if record.Load().Read().Number != 0 {
		t.Fatal("zero record cell changed the zero value")
	}
}

// TestCellStore checks the private root copy and shared immutable backing storage.
func TestCellStore(t *testing.T) {
	input := testFrozen{Number: 1, Slice: FreezeSlice([]int{2}), Child: &testChild{Number: 3}}
	var cell Cell[testFrozen]
	cell.Store(input)
	if cell.value.Load() == &input {
		t.Fatal("cell kept the caller's root address")
	}
	input.Number = 9
	first := cell.Load()
	shared := cell.LoadShared()
	if first.value.Number != 1 || shared.Number != 1 {
		t.Fatal("cell root changed with the caller's value")
	}
	if &shared.Slice.values[0] != &input.Slice.values[0] || shared.Child != input.Child {
		t.Fatal("cell copied immutable backing storage")
	}
	shared.Number = 8
	if cell.LoadShared().Number != 1 {
		t.Fatal("LoadShared returned the cell's root storage")
	}
	cell.Store(testFrozen{Number: 4})
	if first.value.Number != 1 || cell.LoadShared().Number != 4 {
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
				value := Value(cell.Load())
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

// TestCellSharedPointer checks the selected root identity and its lifetime across Store.
func TestCellSharedPointer(t *testing.T) {
	var cell Cell[testFrozen]
	if cell.LoadSharedPointer() != nil {
		t.Fatal("empty Cell returned a pointer")
	}
	cell.Store(testFrozen{Number: 3})
	first := cell.LoadSharedPointer()
	if first != cell.LoadSharedPointer() || first != cell.value.Load() {
		t.Fatal("load copied the selected root")
	}
	cell.Store(testFrozen{Number: 7})
	if first.Number != 3 || first == cell.LoadSharedPointer() || cell.LoadSharedPointer().Number != 7 {
		t.Fatal("Store changed the saved version")
	}
	if got := testing.AllocsPerRun(100, func() { sharedPointerSink = cell.LoadSharedPointer() }); got != 0 {
		t.Fatalf("LoadSharedPointer allocated %g times", got)
	}
}

var sharedPointerSink *testFrozen
