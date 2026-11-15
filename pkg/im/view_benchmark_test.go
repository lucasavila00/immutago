package im

import (
	"strings"
	"testing"
)

var viewRecordSink View[testFrozen]
var viewStringSink View[string]
var viewSliceSink View[FrozenSlice[int]]
var viewMapSink View[FrozenMap[int, int]]
var viewPointerSink View[*testChild]
var viewNumberSink View[int]
var viewArraySink View[[256]int]
var stringViewSink StringView
var sliceViewSink SliceView[int]
var mapViewSink MapView[int, int]
var stringCloneSink string
var cellSharedSink testFrozen
var stringByteSink byte

// TestViewAllocations checks view construction, projection, and adapters without allocations.
func TestViewAllocations(t *testing.T) {
	input := testFrozen{
		Text: strings.Repeat("text", 128), Slice: FreezeSlice([]int{2, 4}),
		Map: FreezeMap(map[int]int{1: 3}), Child: &testChild{Number: 5}, Array: [2]int{6, 7},
	}
	view := ViewOf(input)
	index := 1
	cases := []struct {
		name string
		run  func()
	}{
		{"view", func() { viewRecordSink = ViewOf(input) }},
		{"select string", func() { viewStringSink = Select(view, func(v testFrozen) string { return v.Text }) }},
		{"select slice", func() {
			viewSliceSink = Select(view, func(v testFrozen) FrozenSlice[int] { return v.Slice })
		}},
		{"select map", func() {
			viewMapSink = Select(view, func(v testFrozen) FrozenMap[int, int] { return v.Map })
		}},
		{"select pointer", func() {
			viewPointerSink = Select(view, func(v testFrozen) *testChild { return v.Child })
		}},
		{"select array index", func() {
			viewNumberSink = Select(view, func(v testFrozen) int { return v.Array[index] })
		}},
		{"text adapter", func() { stringViewSink = Text(ViewOf(input.Text)) }},
		{"slice adapter", func() { sliceViewSink = Slice(ViewOf(input.Slice)) }},
		{"map adapter", func() { mapViewSink = Map(ViewOf(input.Map)) }},
		{"copy", func() { benchmarkInt = Copy(ViewOf(input.Number)) }},
		{"nil test", func() { benchmarkFound = IsNil(ViewOf(input.Child)) }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := testing.AllocsPerRun(100, test.run); got != 0 {
				t.Fatalf("allocations = %g; want zero", got)
			}
		})
	}
}

// TestViewReadAllocations checks reads and direct iteration without allocations.
func TestViewReadAllocations(t *testing.T) {
	text := Text(ViewOf(strings.Repeat("text", 128)))
	sequence := Slice(ViewOf(FreezeSlice([]int{2, 4, 6})))
	mapping := Map(ViewOf(FreezeMap(map[int]int{1: 2, 3: 4})))
	cases := []struct {
		name string
		run  func()
	}{
		{"string read", func() { stringByteSink = text.At(1); benchmarkInt = text.Len() }},
		{"slice read", func() { benchmarkInt = Copy(sequence.At(1)) + sequence.Len() }},
		{"map read", func() {
			value, found := mapping.Get(1)
			benchmarkInt, benchmarkFound = Copy(value)+mapping.Len(), found
		}},
		{"slice iteration", func() {
			for index, value := range sequence.All() {
				benchmarkInt = index + Copy(value)
			}
		}},
		{"map iteration", func() {
			for key, value := range mapping.All() {
				benchmarkInt = Copy(key) + Copy(value)
			}
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := testing.AllocsPerRun(100, test.run); got != 0 {
				t.Fatalf("allocations = %g; want zero", got)
			}
		})
	}
}

// TestViewCloneAllocations checks explicit copies whose results escape.
func TestViewCloneAllocations(t *testing.T) {
	text := Text(ViewOf(strings.Repeat("text", 128)))
	sequence := Slice(ViewOf(FreezeSlice([]int{2, 4, 6})))
	mapping := Map(ViewOf(FreezeMap(map[int]int{1: 2, 3: 4})))
	if got := testing.AllocsPerRun(100, func() { stringCloneSink = text.Clone() }); got != 1 {
		t.Fatalf("string clone allocations = %g; want one", got)
	}
	if got := testing.AllocsPerRun(100, func() { benchmarkSliceClone = sequence.Clone() }); got != 1 {
		t.Fatalf("slice clone allocations = %g; want one", got)
	}
	if got := testing.AllocsPerRun(100, func() { benchmarkMapClone = mapping.Clone() }); got == 0 {
		t.Fatal("nonempty map clone did not allocate")
	}
	if got := testing.AllocsPerRun(100, func() { stringCloneSink = (StringView{}).Clone() }); got != 0 {
		t.Fatalf("empty string clone allocations = %g; want zero", got)
	}
	empty := Slice(ViewOf(FreezeSlice(make([]int, 0, 1024))))
	if got := testing.AllocsPerRun(100, func() { benchmarkSliceClone = empty.Clone() }); got != 0 {
		t.Fatalf("empty slice clone allocations = %g; want zero", got)
	}
}

// TestCellAllocations checks private root allocation and reads without allocation.
func TestCellAllocations(t *testing.T) {
	var cell Cell[testFrozen]
	input := testFrozen{Number: 1}
	if got := testing.AllocsPerRun(100, func() { cell.Store(input) }); got != 1 {
		t.Fatalf("cell store allocations = %g; want one", got)
	}
	if got := testing.AllocsPerRun(100, func() { viewRecordSink = cell.Load() }); got != 0 {
		t.Fatalf("cell load allocations = %g; want zero", got)
	}
	if got := testing.AllocsPerRun(100, func() { cellSharedSink = cell.Shared() }); got != 0 {
		t.Fatalf("cell shared allocations = %g; want zero", got)
	}
}

// BenchmarkCellStore measures private root allocation for a snapshot record.
func BenchmarkCellStore(b *testing.B) {
	var cell Cell[testFrozen]
	input := testFrozen{Number: 1}
	b.ReportAllocs()
	for b.Loop() {
		cell.Store(input)
	}
}

// BenchmarkCellLoad measures an atomic load and a root value copy.
func BenchmarkCellLoad(b *testing.B) {
	var cell Cell[testFrozen]
	cell.Store(testFrozen{Number: 1})
	b.ReportAllocs()
	for b.Loop() {
		viewRecordSink = cell.Load()
	}
}

// BenchmarkCellShared measures explicit access to a persistent snapshot.
func BenchmarkCellShared(b *testing.B) {
	var cell Cell[testFrozen]
	cell.Store(testFrozen{Number: 1})
	b.ReportAllocs()
	for b.Loop() {
		cellSharedSink = cell.Shared()
	}
}

// BenchmarkViewOf measures the Go value copy for a root array.
func BenchmarkViewOf(b *testing.B) {
	var input [256]int
	b.ReportAllocs()
	for b.Loop() {
		viewArraySink = ViewOf(input)
	}
}

// BenchmarkSelect measures selection of a string field without a backing copy.
func BenchmarkSelect(b *testing.B) {
	view := ViewOf(testFrozen{Text: strings.Repeat("text", 128)})
	b.ReportAllocs()
	for b.Loop() {
		viewStringSink = Select(view, func(value testFrozen) string { return value.Text })
	}
}

// BenchmarkViewClones measures the explicit allocation cost of native copies.
func BenchmarkViewClones(b *testing.B) {
	text := Text(ViewOf(strings.Repeat("x", 1024)))
	sequence := Slice(ViewOf(FreezeSlice(make([]int, 1024))))
	mapping := Map(ViewOf(FreezeMap(map[int]int{1: 2, 3: 4})))
	b.Run("string", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			stringCloneSink = text.Clone()
		}
	})
	b.Run("slice", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchmarkSliceClone = sequence.Clone()
		}
	})
	b.Run("map", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchmarkMapClone = mapping.Clone()
		}
	})
}
