package im

import (
	"maps"
	"strings"
	"testing"
)

var byValueSliceSink FrozenByValueSlice[int]
var byValueMapSink FrozenByValueMap[int, int]
var stringMapCloneSink map[FrozenString]int

// TestByValueAllocations checks allocation-free construction, views, reads, and direct iteration.
func TestByValueAllocations(t *testing.T) {
	sequence := []int{1, 2, 3, 4}
	entries := map[int]int{1: 2, 3: 4}
	slice := FreezeSlice(sequence)
	mapping := FreezeMap(entries)
	cases := []struct {
		name string
		run  func()
	}{
		{"freeze slice", func() { byValueSliceSink = FreezeByValueSlice(sequence) }},
		{"freeze map", func() { byValueMapSink = FreezeByValueMap(entries) }},
		{"slice view", func() { byValueSliceSink = slice.ByValue() }},
		{"map view", func() { byValueMapSink = mapping.ByValue() }},
		{"slice read", func() { s := slice.ByValue(); benchmarkInt = s.Len() + s.At(1) }},
		{"map read", func() {
			m := mapping.ByValue()
			benchmarkInt, benchmarkFound = m.Get(1)
			benchmarkInt += m.Len()
		}},
		{"slice iteration", func() {
			for index, value := range slice.ByValue().All() {
				benchmarkInt = index + value
			}
		}},
		{"map iteration", func() {
			for key, value := range mapping.ByValue().All() {
				benchmarkInt = key + value
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
	if got := testing.AllocsPerRun(100, func() { benchmarkSliceClone = slice.ByValue().Clone() }); got != 1 {
		t.Fatalf("slice clone allocations = %g; want one", got)
	}
	if got := testing.AllocsPerRun(100, func() { benchmarkMapClone = mapping.ByValue().Clone() }); got == 0 {
		t.Fatal("escaping map clone did not allocate independent storage")
	}
}

// BenchmarkFreezeByValueSlice measures construction of a view from an existing slice.
func BenchmarkFreezeByValueSlice(b *testing.B) {
	input := make([]int, 1024)
	b.ReportAllocs()
	for b.Loop() {
		byValueSliceSink = FreezeByValueSlice(input)
	}
}

// BenchmarkFreezeByValueMap measures construction of a view from an existing map.
func BenchmarkFreezeByValueMap(b *testing.B) {
	input := map[int]int{1: 2}
	b.ReportAllocs()
	for b.Loop() {
		byValueMapSink = FreezeByValueMap(input)
	}
}

// BenchmarkFreezeByValueString measures construction of a view from an existing string.
func BenchmarkFreezeByValueString(b *testing.B) {
	input := strings.Repeat("x", 1024)
	b.ReportAllocs()
	for b.Loop() {
		byValueStringSink = FreezeByValueString(input)
	}
}

// BenchmarkSliceByValue measures conversion to a local slice view.
func BenchmarkSliceByValue(b *testing.B) {
	frozen := FreezeSlice(make([]int, 1024))
	b.ReportAllocs()
	for b.Loop() {
		byValueSliceSink = frozen.ByValue()
	}
}

// BenchmarkMapByValue measures conversion to a local map view.
func BenchmarkMapByValue(b *testing.B) {
	frozen := FreezeMap(map[int]int{1: 2})
	b.ReportAllocs()
	for b.Loop() {
		byValueMapSink = frozen.ByValue()
	}
}

// BenchmarkByValueSliceClone measures explicit storage copying from a local slice view.
func BenchmarkByValueSliceClone(b *testing.B) {
	view := FreezeByValueSlice(make([]int, 1024))
	b.ReportAllocs()
	for b.Loop() {
		benchmarkSliceClone = view.Clone()
	}
}

// BenchmarkByValueMapClone compares local Clone with the same native map copy.
func BenchmarkByValueMapClone(b *testing.B) {
	for _, size := range []struct {
		name  string
		count int
	}{{"small", 2}, {"large", 1024}} {
		input := make(map[int]int, size.count)
		for index := range size.count {
			input[index] = index
		}
		view := FreezeByValueMap(input)
		b.Run(size.name, func(b *testing.B) {
			b.Run("view", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					benchmarkMapClone = view.Clone()
				}
			})
			b.Run("native", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					benchmarkMapClone = maps.Clone(input)
				}
			})
		})
	}
}

// BenchmarkByValueMapStringClone measures map copying with an independent string key.
func BenchmarkByValueMapStringClone(b *testing.B) {
	key := FreezeString(strings.Repeat("x", 1024))
	view := FreezeByValueMap(map[FrozenString]int{key: 1})
	b.ReportAllocs()
	for b.Loop() {
		stringMapCloneSink = view.Clone()
	}
}
