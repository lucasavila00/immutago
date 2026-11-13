package immutable

import (
	"strconv"
	"testing"
)

var benchmarkSlice Slice[int]
var benchmarkMap Map[int, int]
var benchmarkInt int
var benchmarkFound bool

// BenchmarkSliceAt measures allocation-free indexed reads at several tree depths.
func BenchmarkSliceAt(b *testing.B) {
	for _, size := range []int{32, 1024, 32768} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			values := NewSlice(make([]int, size)...)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				benchmarkInt = values.At(size / 2)
			}
		})
	}
}

// BenchmarkSliceSet measures a point update while retaining the original sequence.
func BenchmarkSliceSet(b *testing.B) {
	for _, size := range []int{32, 1024, 32768} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			values := NewSlice(make([]int, size)...)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				benchmarkSlice = values.Set(size/2, 1)
			}
		})
	}
}

// BenchmarkSliceAppend measures append across a full tail without copying existing blocks.
func BenchmarkSliceAppend(b *testing.B) {
	for _, size := range []int{32, 1024, 32768} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			values := NewSlice(make([]int, size)...)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				benchmarkSlice = values.Append(1)
			}
		})
	}
}

// BenchmarkMapGet measures allocation-free lookup at several map sizes.
func BenchmarkMapGet(b *testing.B) {
	for _, size := range []int{32, 1024, 32768} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			values := benchmarkMapOfSize(size)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				benchmarkInt, benchmarkFound = values.Get(size / 2)
			}
		})
	}
}

// BenchmarkMapSet measures a replacement while retaining the original map.
func BenchmarkMapSet(b *testing.B) {
	for _, size := range []int{32, 1024, 32768} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			values := benchmarkMapOfSize(size)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				benchmarkMap = values.Set(size/2, 1)
			}
		})
	}
}

// benchmarkMapOfSize constructs integer keys outside the timed benchmark region.
func benchmarkMapOfSize(size int) Map[int, int] {
	var values Map[int, int]
	for key := 0; key < size; key++ {
		values = values.Set(key, key)
	}
	return values
}
