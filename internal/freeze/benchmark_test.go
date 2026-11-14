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
				graph, effects := benchmarkFlow(count, topology)
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					consumedFlow(graph, effects, count)
				}
			})
		}
	}
}

// benchmarkFlow creates a graph that retains each consumed owner at later points.
func benchmarkFlow(count int, topology string) (flow.Graph, []ownerEffect) {
	successors := make([][]int, count+1)
	effects := make([]ownerEffect, count+1)
	for point := range count + 1 {
		if point < count {
			effects[point] = ownerEffect{owner: point, consume: true}
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
	return flow.NewGraph(successors), effects
}
