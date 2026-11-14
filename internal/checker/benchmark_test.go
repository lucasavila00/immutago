package checker

import (
	"fmt"
	"testing"
)

// BenchmarkCoreReason measures complete proofs of valid and invalid type graphs.
func BenchmarkCoreReason(b *testing.B) {
	for _, shape := range []string{"Chain", "Fanout", "Cycle"} {
		for _, size := range []int{16, 128, 1024} {
			for _, valid := range []bool{true, false} {
				b.Run(fmt.Sprintf("%s/%d/valid=%t", shape, size, valid), func(b *testing.B) {
					benchmarkCoreReasonGraph(b, shape, size, valid)
				})
			}
		}
	}
}

// benchmarkCoreReasonGraph prepares one graph before timing repeated proofs.
func benchmarkCoreReasonGraph(b *testing.B, shape string, size int, valid bool) {
	b.Helper()
	bad := -1
	if !valid {
		bad = size - 1
	}
	root := stressGraph(stressEdges(size, shape), bad, false)[0]
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if got := Reason(root) == ""; got != valid {
			b.Fatalf("valid=%t, want %t", got, valid)
		}
	}
}

// BenchmarkCoreContainsCold measures a fresh classifier on unmarked recursive storage.
func BenchmarkCoreContainsCold(b *testing.B) {
	for _, size := range []int{16, 128, 1024} {
		for _, protected := range []bool{false, true} {
			b.Run(fmt.Sprintf("Cycle/%d/protected=%t", size, protected), func(b *testing.B) {
				root := stressContainsGraph(size, protected)
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					if got := Contains(root); got != protected {
						b.Fatalf("contains=%t, want %t", got, protected)
					}
				}
			})
		}
	}
}
