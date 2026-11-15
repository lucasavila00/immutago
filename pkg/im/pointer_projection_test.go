package im

import "testing"

var pointerProjectionSink Projection[*testFrozen]
var pointerNumberSink int

// TestPointerProjections checks stable root and element addresses across publication.
func TestPointerProjections(t *testing.T) {
	var cell Cell[testFrozen]
	if !IsNil(cell.LoadPointer()) {
		t.Fatal("empty cell must return a nil pointer projection")
	}
	cell.Store(testFrozen{Number: 3, Array: [2]int{4, 5}})
	first := cell.LoadPointer()
	if first.value != cell.value.Load() || first.Read().Number != 3 {
		t.Fatal("pointer read changed the stored root")
	}
	cell.Store(testFrozen{Number: 9})
	if first.value.Number != 3 || cell.LoadPointer().value.Number != 9 {
		t.Fatal("publication changed an earlier pointer projection")
	}
	values := FreezeSlice([]testChild{{Number: 6}})
	element := Slice(ProjectionOf(values)).AtPointer(0)
	if element.value != &values.values[0] || element.value.Number != 6 {
		t.Fatal("element pointer copied or changed the element")
	}
}

// TestCopyProjection checks one root copy without a recursive backing-storage copy.
func TestCopyProjection(t *testing.T) {
	input := testFrozen{Number: 3, Slice: FreezeSlice([]int{4}), Child: &testChild{Number: 5}}
	projection := CopyProjection(ProjectionOf(input))
	input.Number = 9
	if projection.value.Number != 3 || projection.value == &input {
		t.Fatal("CopyProjection retained the caller's root")
	}
	if &projection.value.Slice.values[0] != &input.Slice.values[0] || projection.value.Child != input.Child {
		t.Fatal("CopyProjection copied immutable backing storage")
	}
}

// TestPointerProjectionAllocations checks the read path and the explicit root-copy cost.
func TestPointerProjectionAllocations(t *testing.T) {
	var cell Cell[testFrozen]
	cell.Store(testFrozen{Number: 3})
	values := Slice(ProjectionOf(FreezeSlice([]testChild{{Number: 4}})))
	if got := testing.AllocsPerRun(100, func() {
		pointerProjectionSink = cell.LoadPointer()
		pointerNumberSink = pointerProjectionSink.Read().Number
		pointerNumberSink += values.AtPointer(0).Read().Number
	}); got != 0 {
		t.Fatalf("pointer reads allocated %g times; want 0", got)
	}
	input := ProjectionOf(testFrozen{Number: 3})
	if got := testing.AllocsPerRun(100, func() { pointerProjectionSink = CopyProjection(input) }); got != 1 {
		t.Fatalf("CopyProjection allocated %g times; want 1", got)
	}
}

type largePointerRoot struct {
	Frozen
	Number int
	Values [8192]int
}

// BenchmarkPointerProjectionRead compares an address-backed read with the native atomic path.
func BenchmarkPointerProjectionRead(b *testing.B) {
	var cell Cell[largePointerRoot]
	cell.Store(largePointerRoot{Number: 3})
	b.Run("Native", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			pointerNumberSink = cell.value.Load().Number
		}
	})
	b.Run("Projection", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			pointerNumberSink = cell.LoadPointer().Read().Number
		}
	})
}
