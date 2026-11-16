package im

import (
	"strings"
	"testing"
)

var projectionRecordSink Projection[testFrozen]
var projectionStringSink Projection[string]
var projectionSliceSink Projection[FrozenSlice[int]]
var projectionMapSink Projection[FrozenMap[int, int]]
var projectionPointerSink Projection[*testChild]
var projectionNumberSink Projection[int]
var projectionArraySink Projection[[256]int]
var stringProjectionSink StringProjection
var sliceProjectionSink SliceProjection[int]
var mapProjectionSink MapProjection[int, int]
var stringCloneSink string
var cellSharedSink testFrozen
var stringByteSink byte

// TestProjectionAllocations checks projection construction, projection, and adapters without allocations.
func TestProjectionAllocations(t *testing.T) {
	input := testFrozen{
		Text: strings.Repeat("text", 128), Slice: FreezeSlice([]int{2, 4}),
		Map: FreezeMap(map[int]int{1: 3}), Child: &testChild{Number: 5}, Array: [2]int{6, 7},
	}
	projection := ProjectionOf(input)
	index := 1
	cases := []struct {
		name string
		run  func()
	}{
		{"projection", func() { projectionRecordSink = ProjectionOf(input) }},
		{"read string", func() { projectionStringSink = ProjectionOf(projection.Unchecked().Text) }},
		{"read slice", func() {
			projectionSliceSink = ProjectionOf(projection.Unchecked().Slice)
		}},
		{"read map", func() {
			projectionMapSink = ProjectionOf(projection.Unchecked().Map)
		}},
		{"read pointer", func() {
			projectionPointerSink = ProjectionOf(projection.Unchecked().Child)
		}},
		{"read array index", func() {
			projectionNumberSink = ProjectionOf(projection.Unchecked().Array[index])
		}},
		{"native string", func() { stringProjectionSink = ProjectString(input.Text) }},
		{"frozen slice", func() { sliceProjectionSink = input.Slice.Projection() }},
		{"frozen map", func() { mapProjectionSink = input.Map.Projection() }},
		{"text adapter", func() { stringProjectionSink = StringOf(ProjectionOf(input.Text)) }},
		{"slice adapter", func() { sliceProjectionSink = SliceOf(ProjectionOf(input.Slice)) }},
		{"map adapter", func() { mapProjectionSink = MapOf(ProjectionOf(input.Map)) }},
		{"value", func() { benchmarkInt = Value(ProjectionOf(input.Number)) }},
		{"nil test", func() { benchmarkFound = IsNil(ProjectionOf(input.Child)) }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := testing.AllocsPerRun(100, test.run); got != 0 {
				t.Fatalf("allocations = %g; want zero", got)
			}
		})
	}
}

// TestProjectionReadAllocations checks reads and direct iteration without allocations.
func TestProjectionReadAllocations(t *testing.T) {
	text := StringOf(ProjectionOf(strings.Repeat("text", 128)))
	sequence := SliceOf(ProjectionOf(FreezeSlice([]int{2, 4, 6})))
	mapping := MapOf(ProjectionOf(FreezeMap(map[int]int{1: 2, 3: 4})))
	cases := []struct {
		name string
		run  func()
	}{
		{"string read", func() { stringByteSink = text.At(1); benchmarkInt = text.Len() }},
		{"slice read", func() { benchmarkInt = Value(sequence.At(1)) + sequence.Len() }},
		{"map read", func() {
			value, found := mapping.Get(1)
			benchmarkInt, benchmarkFound = Value(value)+mapping.Len(), found
		}},
		{"slice iteration", func() {
			for index, value := range sequence.All() {
				benchmarkInt = index + Value(value)
			}
		}},
		{"map iteration", func() {
			for key, value := range mapping.All() {
				benchmarkInt = Value(key) + Value(value)
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

// TestProjectionCloneAllocations checks explicit copies whose results escape.
func TestProjectionCloneAllocations(t *testing.T) {
	text := StringOf(ProjectionOf(strings.Repeat("text", 128)))
	sequence := SliceOf(ProjectionOf(FreezeSlice([]int{2, 4, 6})))
	mapping := MapOf(ProjectionOf(FreezeMap(map[int]int{1: 2, 3: 4})))
	if got := testing.AllocsPerRun(100, func() { stringCloneSink = text.Clone() }); got != 1 {
		t.Fatalf("string clone allocations = %g; want one", got)
	}
	if got := testing.AllocsPerRun(100, func() { benchmarkSliceClone = sequence.Clone() }); got != 1 {
		t.Fatalf("slice clone allocations = %g; want one", got)
	}
	if got := testing.AllocsPerRun(100, func() { benchmarkMapClone = mapping.Clone() }); got == 0 {
		t.Fatal("nonempty map clone did not allocate")
	}
	if got := testing.AllocsPerRun(100, func() { stringCloneSink = (StringProjection{}).Clone() }); got != 0 {
		t.Fatalf("empty string clone allocations = %g; want zero", got)
	}
	empty := SliceOf(ProjectionOf(FreezeSlice(make([]int, 0, 1024))))
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
	if got := testing.AllocsPerRun(100, func() { projectionRecordSink = cell.Load() }); got != 0 {
		t.Fatalf("cell load allocations = %g; want zero", got)
	}
	if got := testing.AllocsPerRun(100, func() { cellSharedSink = cell.Load().Unchecked() }); got != 0 {
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
		projectionRecordSink = cell.Load()
	}
}

// BenchmarkCellUncheckedValue measures explicit access to a persistent snapshot.
func BenchmarkCellUncheckedValue(b *testing.B) {
	var cell Cell[testFrozen]
	cell.Store(testFrozen{Number: 1})
	b.ReportAllocs()
	for b.Loop() {
		cellSharedSink = cell.Load().Unchecked()
	}
}

// BenchmarkProjectionOf measures the Go value copy for a root array.
func BenchmarkProjectionOf(b *testing.B) {
	var input [256]int
	b.ReportAllocs()
	for b.Loop() {
		projectionArraySink = ProjectionOf(input)
	}
}

// BenchmarkProjectionField measures selection of a string field without a backing copy.
func BenchmarkProjectionField(b *testing.B) {
	projection := ProjectionOf(&testFrozen{Text: strings.Repeat("text", 128)})
	b.ReportAllocs()
	for b.Loop() {
		projectionStringSink = ProjectionOf(projection.Unchecked().Text)
	}
}

// BenchmarkProjectionClones measures the explicit allocation cost of native copies.
func BenchmarkProjectionClones(b *testing.B) {
	text := StringOf(ProjectionOf(strings.Repeat("x", 1024)))
	sequence := SliceOf(ProjectionOf(FreezeSlice(make([]int, 1024))))
	mapping := MapOf(ProjectionOf(FreezeMap(map[int]int{1: 2, 3: 4})))
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
