package analyzer

import (
	"encoding/gob"
	"fmt"
	"io"
	"testing"
)

// BenchmarkCoreDependencyFacts measures merge, encoding, and reporting after graph setup.
func BenchmarkCoreDependencyFacts(b *testing.B) {
	for _, size := range []int{4, 12, 20} {
		facts := layeredDependencyFacts(size, size, 101)
		last := facts[len(facts)-1]
		path := fmt.Sprintf("p%02d_%02d", size-1, size-1)
		imports := make(map[string]invalidPackage)
		for column, fact := range facts[len(facts)-size:] {
			imports[fmt.Sprintf("p%02d_%02d", size-1, column)] = *fact
		}
		b.Run(fmt.Sprintf("merge/layers=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = packageFailureFact("consumer", false, imports, nil)
			}
		})
		b.Run(fmt.Sprintf("gob/layers=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if err := gob.NewEncoder(io.Discard).Encode(last); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(fmt.Sprintf("report/layers=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = dependencyMessage("consumer", path, *last)
			}
		})
	}
}
