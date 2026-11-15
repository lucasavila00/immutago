package im

import "testing"

var pointerViewSink View[*testFrozen]
var pointerNumberSink int

// TestPointerViews checks stable root and element addresses across publication.
func TestPointerViews(t *testing.T) {
	var cell Cell[testFrozen]
	if !IsNil(cell.LoadPointer()) {
		t.Fatal("empty cell must return a nil pointer view")
	}
	cell.Store(testFrozen{Number: 3, Array: [2]int{4, 5}})
	first := cell.LoadPointer()
	if first.value != cell.value.Load() || Copy(Select(first, func(v *testFrozen) int { return v.Number })) != 3 {
		t.Fatal("pointer read changed the stored root")
	}
	cell.Store(testFrozen{Number: 9})
	if first.value.Number != 3 || cell.LoadPointer().value.Number != 9 {
		t.Fatal("publication changed an earlier pointer view")
	}
	values := FreezeSlice([]testChild{{Number: 6}})
	element := Slice(ViewOf(values)).AtPointer(0)
	if element.value != &values.values[0] || element.value.Number != 6 {
		t.Fatal("element pointer copied or changed the element")
	}
}

// TestCopyView checks one root copy without a recursive backing-storage copy.
func TestCopyView(t *testing.T) {
	input := testFrozen{Number: 3, Slice: FreezeSlice([]int{4}), Child: &testChild{Number: 5}}
	view := CopyView(ViewOf(input))
	input.Number = 9
	if view.value.Number != 3 || view.value == &input {
		t.Fatal("CopyView retained the caller's root")
	}
	if &view.value.Slice.values[0] != &input.Slice.values[0] || view.value.Child != input.Child {
		t.Fatal("CopyView copied immutable backing storage")
	}
}

// TestPointerViewAllocations checks the read path and the explicit root-copy cost.
func TestPointerViewAllocations(t *testing.T) {
	var cell Cell[testFrozen]
	cell.Store(testFrozen{Number: 3})
	values := Slice(ViewOf(FreezeSlice([]testChild{{Number: 4}})))
	if got := testing.AllocsPerRun(100, func() {
		pointerViewSink = cell.LoadPointer()
		pointerNumberSink = Copy(Select(pointerViewSink, func(v *testFrozen) int { return v.Number }))
		pointerNumberSink += Copy(Select(values.AtPointer(0), func(v *testChild) int { return v.Number }))
	}); got != 0 {
		t.Fatalf("pointer reads allocated %g times; want 0", got)
	}
	input := ViewOf(testFrozen{Number: 3})
	if got := testing.AllocsPerRun(100, func() { pointerViewSink = CopyView(input) }); got != 1 {
		t.Fatalf("CopyView allocated %g times; want 1", got)
	}
}

type largePointerRoot struct {
	Frozen
	Number int
	Values [8192]int
}

// BenchmarkPointerViewRead compares an address-backed read with the native atomic path.
func BenchmarkPointerViewRead(b *testing.B) {
	var cell Cell[largePointerRoot]
	cell.Store(largePointerRoot{Number: 3})
	b.Run("Native", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			pointerNumberSink = cell.value.Load().Number
		}
	})
	b.Run("View", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			pointerNumberSink = Copy(Select(cell.LoadPointer(), func(v *largePointerRoot) int { return v.Number }))
		}
	})
}
