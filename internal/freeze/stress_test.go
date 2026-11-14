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
		graph, effects := stressFlow(seed, 48, owners)
		got := consumedFlow(graph, effects, owners)
		for owner := range owners {
			want := consumedPaths(graph, effects, owner)
			for point := range graph.Succ {
				if got.has(point, owner) != want[point] {
					t.Fatalf("seed %d, point %d, owner %d: got %v, want %v", seed, point, owner, got.has(point, owner), want[point])
				}
			}
		}
	}
}

// stressFlow makes deterministic branches, cycles, and unreachable points.
func stressFlow(seed int64, points, owners int) (flow.Graph, []ownerEffect) {
	random := rand.New(rand.NewSource(seed))
	successors := make([][]int, points)
	effects := make([]ownerEffect, points)
	for point := range points {
		// Keep the last four points unreachable, including edges into the graph.
		limit := points - 4
		for range random.Intn(4) {
			successors[point] = append(successors[point], random.Intn(limit))
		}
		if point < limit-1 && random.Intn(3) != 0 {
			successors[point] = append(successors[point], point+1)
		}
		kind := random.Intn(3)
		effects[point] = ownerEffect{owner: random.Intn(owners), consume: kind == 1, reset: kind == 2}
	}
	return flow.NewGraph(successors), effects
}

// consumedPaths follows each possible consumed state from the entry point.
func consumedPaths(graph flow.Graph, effects []ownerEffect, owner int) []bool {
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
		out := current.consumed
		effect := effects[current.point]
		if effect.owner == owner {
			out = (out && !effect.reset) || effect.consume
		}
		for _, next := range graph.Succ[current.point] {
			queue = append(queue, state{next, out})
		}
	}
	return want
}

// TestConsumedFlowWordBoundaries checks owners across dense word boundaries.
func TestConsumedFlowWordBoundaries(t *testing.T) {
	effects := []ownerEffect{
		{owner: 0, consume: true}, {owner: 63, consume: true}, {owner: 64, consume: true},
		{owner: 65, consume: true}, {owner: 1023, consume: true}, {},
		{owner: 63, reset: true}, {owner: 64, reset: true}, {owner: 65, reset: true},
		{owner: 1023, reset: true}, {owner: 65, consume: true}, {},
		{owner: 1, consume: true}, // Unreachable, with an edge into the reachable graph.
	}
	successors := make([][]int, len(effects))
	for point := 0; point < 11; point++ {
		successors[point] = []int{point + 1}
	}
	successors[12] = []int{11}
	got := consumedFlow(flow.NewGraph(successors), effects, 1024)
	for _, owner := range []int{0, 63, 64, 65, 1023} {
		if !got.has(5, owner) {
			t.Errorf("owner %d missing before reset", owner)
		}
		want := owner == 0 || owner == 65
		if got.has(11, owner) != want {
			t.Errorf("owner %d: state after reset does not match %v", owner, want)
		}
	}
	if got.has(11, 1) || got.has(12, 0) {
		t.Error("unreachable point changed the result")
	}
}
