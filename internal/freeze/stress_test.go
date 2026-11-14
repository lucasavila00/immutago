package freeze

import (
	"math/rand"
	"testing"

	"github.com/lucasavila00/immutago/internal/freeze/flow"
)

// TestConsumedFlowStress compares the dataflow result with individual owner paths.
func TestConsumedFlowStress(t *testing.T) {
	for seed := int64(0); seed < 80; seed++ {
		const owners = 12
		graph, gen, kill := stressFlow(seed, 48, owners)
		got := consumedFlow(graph, gen, kill, owners)
		for owner := range owners {
			want := consumedPaths(graph, gen, kill, owner)
			for point := range graph.Succ {
				if got.has(point, owner) != want[point] {
					t.Fatalf("seed %d, point %d, owner %d: got %v, want %v", seed, point, owner, got.has(point, owner), want[point])
				}
			}
		}
	}
}

// stressFlow makes deterministic branches, cycles, and unreachable points.
func stressFlow(seed int64, points, owners int) (flow.Graph, []flow.Set, []flow.Set) {
	random := rand.New(rand.NewSource(seed))
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
	return flow.NewGraph(successors), gen, kill
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

// TestConsumedFlowWordBoundaries checks owners across dense word boundaries.
func TestConsumedFlowWordBoundaries(t *testing.T) {
	graph := flow.NewGraph([][]int{{1}, {2}, {3}, nil, {3}})
	gen := []flow.Set{flow.NewSet(0, 63, 64, 65, 1023), flow.NewSet(65), nil, nil, flow.NewSet(1)}
	kill := []flow.Set{nil, flow.NewSet(63, 64, 65, 1023), nil, nil, nil}
	got := consumedFlow(graph, gen, kill, 1024)
	for _, owner := range []int{0, 63, 64, 65, 1023} {
		if !got.has(1, owner) {
			t.Errorf("owner %d missing before reset", owner)
		}
		want := owner == 0 || owner == 65
		if got.has(2, owner) != want || got.has(3, owner) != want {
			t.Errorf("owner %d: state after reset does not match %v", owner, want)
		}
	}
	if got.has(3, 1) || got.has(4, 0) {
		t.Error("unreachable point changed the result")
	}
}
