package freeze

import (
	"math/rand"
	"testing"

	"github.com/lucasavila00/immutago/internal/freeze/flow"
)

// TestConsumedFlowStress compares the dataflow result with individual owner paths.
func TestConsumedFlowStress(t *testing.T) {
	for seed := int64(0); seed < 80; seed++ {
		random := rand.New(rand.NewSource(seed))
		const points, owners = 48, 12
		successors := make([][]int, points)
		gen, kill := make([]flow.Set, points), make([]flow.Set, points)
		for point := range points {
			gen[point], kill[point] = flow.NewSet(), flow.NewSet()
			// Keep the last four points unreachable, including edges into the graph.
			limit := points - 4
			for range random.Intn(4) {
				successors[point] = append(successors[point], random.Intn(limit))
			}
			if point < limit-1 && random.Intn(3) != 0 {
				successors[point] = append(successors[point], point+1)
			}
			for owner := range owners {
				if random.Intn(12) == 0 {
					gen[point].Add(owner)
				}
				if random.Intn(12) == 0 {
					kill[point].Add(owner)
				}
			}
		}
		graph := flow.NewGraph(successors)
		got := consumedFlow(graph, gen, kill)
		for owner := range owners {
			want := consumedPaths(graph, gen, kill, owner)
			for point := range points {
				if got[point][owner] != want[point] {
					t.Fatalf("seed %d, point %d, owner %d: got %v, want %v", seed, point, owner, got[point][owner], want[point])
				}
			}
		}
	}
}

// consumedPaths follows each possible consumed state from the entry point.
func consumedPaths(graph flow.Graph, gen, kill []flow.Set, owner int) []bool {
	type state struct {
		point    int
		consumed bool
	}
	want := make([]bool, len(graph.Succ))
	seen := make(map[state]bool)
	queue := []state{{}}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if seen[current] {
			continue
		}
		seen[current] = true
		want[current.point] = want[current.point] || current.consumed
		out := (current.consumed && !kill[current.point][owner]) || gen[current.point][owner]
		for _, next := range graph.Succ[current.point] {
			queue = append(queue, state{next, out})
		}
	}
	return want
}
