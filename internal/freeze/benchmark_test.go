package freeze

import (
	"fmt"
	"testing"

	"github.com/lucasavila00/immutago/internal/freeze/flow"
)

// BenchmarkCoreConsumedFlow measures many owners in linear and cyclic graphs.
func BenchmarkCoreConsumedFlow(b *testing.B) {
	for _, count := range []int{64, 256, 1024} {
		for _, topology := range []string{"linear", "branches", "cycle"} {
			name := fmt.Sprintf("owners=%d/%s", count, topology)
			b.Run(name, func(b *testing.B) {
				graph, gen, kill := benchmarkFlow(count, topology)
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					consumedFlow(graph, gen, kill, count)
				}
			})
		}
	}
}

// benchmarkFlow creates a graph that retains each consumed owner at later points.
func benchmarkFlow(count int, topology string) (flow.Graph, []flow.Set, []flow.Set) {
	successors := make([][]int, count+1)
	gen, kill := make([]flow.Set, count+1), make([]flow.Set, count+1)
	for point := range count + 1 {
		gen[point], kill[point] = flow.NewSet(), flow.NewSet()
		if point < count {
			gen[point].Add(point)
			successors[point] = append(successors[point], point+1)
		}
	}
	if topology == "branches" {
		for point := 0; point+2 <= count; point += 2 {
			successors[point] = append(successors[point], point+2)
		}
	}
	if topology == "cycle" {
		successors[count] = append(successors[count], count/2)
	}
	return flow.NewGraph(successors), gen, kill
}
