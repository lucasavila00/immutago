package im

import "testing"

var benchmarkSlice FrozenSlice[int]
var benchmarkMap FrozenMap[int, int]
var benchmarkInt int
var benchmarkFound bool
var benchmarkSliceClone []int
var benchmarkMapClone map[int]int

// BenchmarkFreezeSlice measures allocation-free wrapping of an existing backing array.
func BenchmarkFreezeSlice(b *testing.B) {
	input := make([]int, 1024)
	b.ReportAllocs()
	for b.Loop() {
		benchmarkSlice = FreezeSlice(input)
	}
}

// BenchmarkFreezeMap measures allocation-free wrapping of an existing native map.
func BenchmarkFreezeMap(b *testing.B) {
	input := make(map[int]int, 1024)
	for i := range 1024 {
		input[i] = i
	}
	b.ReportAllocs()
	for b.Loop() {
		benchmarkMap = FreezeMap(input)
	}
}

// BenchmarkSliceAt measures native indexed reads without allocations.
func BenchmarkSliceAt(b *testing.B) {
	frozen := FreezeSlice(make([]int, 1024))
	b.ReportAllocs()
	for b.Loop() {
		benchmarkInt = frozen.At(512)
	}
}

// BenchmarkMapGet measures native key lookup without allocations.
func BenchmarkMapGet(b *testing.B) {
	frozen := FreezeMap(map[int]int{1: 2})
	b.ReportAllocs()
	for b.Loop() {
		benchmarkInt, benchmarkFound = frozen.Get(1)
	}
}

// BenchmarkSliceClone measures the explicit cost of independent mutable storage.
func BenchmarkSliceClone(b *testing.B) {
	frozen := FreezeSlice(make([]int, 1024))
	b.ReportAllocs()
	for b.Loop() {
		benchmarkSliceClone = frozen.Clone()
	}
}

// BenchmarkMapClone measures the explicit cost of independent mutable map entries.
func BenchmarkMapClone(b *testing.B) {
	input := make(map[int]int, 1024)
	for i := range 1024 {
		input[i] = i
	}
	frozen := FreezeMap(input)
	b.ReportAllocs()
	for b.Loop() {
		benchmarkMapClone = frozen.Clone()
	}
}

// BenchmarkSliceAll measures direct traversal without copying storage.
func BenchmarkSliceAll(b *testing.B) {
	frozen := FreezeSlice(make([]int, 1024))
	b.ReportAllocs()
	for b.Loop() {
		for index, value := range frozen.All() {
			benchmarkInt = index + value
		}
	}
}

// BenchmarkMapAll measures direct map traversal without copying entries.
func BenchmarkMapAll(b *testing.B) {
	input := make(map[int]int, 1024)
	for i := range 1024 {
		input[i] = i
	}
	frozen := FreezeMap(input)
	b.ReportAllocs()
	for b.Loop() {
		for key, value := range frozen.All() {
			benchmarkInt = key + value
		}
	}
}
