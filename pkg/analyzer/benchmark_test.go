package analyzer

import (
	"fmt"
	"testing"

	"github.com/lucasavila00/immutago/internal/checker"
	"golang.org/x/tools/go/analysis"
)

// BenchmarkCoreMatcher measures source matching without parsing or Go type-checking setup.
func BenchmarkCoreMatcher(b *testing.B) {
	for _, shape := range []string{"records", "wide", "nested"} {
		for _, count := range []int{16, 64, 256} {
			b.Run(fmt.Sprintf("%s/%d", shape, count), func(b *testing.B) {
				pass := sourcePass(b, stressSource(shape, count, false))
				pass.Report = func(d analysis.Diagnostic) { b.Fatalf("valid source rejected: %s", d.Message) }
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					checkSubset(pass, nil, new(checker.Prover))
				}
			})
		}
	}
}

// BenchmarkCoreAnalyzer measures the full package pass for many separate builders in one function.
func BenchmarkCoreAnalyzer(b *testing.B) {
	for _, count := range []int{16, 64, 256} {
		b.Run(fmt.Sprintf("builders/%d", count), func(b *testing.B) {
			pass := sourcePass(b, stressSource("builders", count, false))
			pass.Report = func(d analysis.Diagnostic) { b.Fatalf("valid source rejected: %s", d.Message) }
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if _, err := run(pass); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
