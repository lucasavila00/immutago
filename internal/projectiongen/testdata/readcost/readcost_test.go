package readcost

import (
	"sync/atomic"
	"testing"
)

// Native roots have the same field layout without a protected marker.
type native8KiB struct {
	_       struct{}
	Padding [8192]byte
	Count   int
}

type native64KiB struct {
	_       struct{}
	Padding [65536]byte
	Count   int
}

var native8 atomic.Pointer[native8KiB]
var native64 atomic.Pointer[native64KiB]
var readCount int

// BenchmarkGenerated8KiB measures publication selection and one generated field read.
func BenchmarkGenerated8KiB(b *testing.B) {
	record8.Store(Record8KiB{Count: 1})
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		readCount = record8.Load().Count()
	}
}

// BenchmarkNative8KiB measures the same read through a native atomic pointer.
func BenchmarkNative8KiB(b *testing.B) {
	native8.Store(&native8KiB{Count: 1})
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		readCount = native8.Load().Count
	}
}

// BenchmarkGenerated64KiB repeats the generated read with a larger root.
func BenchmarkGenerated64KiB(b *testing.B) {
	record64.Store(Record64KiB{Count: 1})
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		readCount = record64.Load().Count()
	}
}

// BenchmarkNative64KiB supplies the native comparison for the larger root.
func BenchmarkNative64KiB(b *testing.B) {
	native64.Store(&native64KiB{Count: 1})
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		readCount = native64.Load().Count
	}
}
