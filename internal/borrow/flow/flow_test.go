package flow

import (
	"maps"
	"math/rand/v2"
	"slices"
	"testing"
)

// TestBackwardLive traces branch and loop uses to definitions, not textual scope ends.
func TestBackwardLive(t *testing.T) {
	graph := NewGraph([][]int{{1}, {2, 4}, {3}, {1}, {5}, nil})
	requireSet(t, BackwardLive(graph, NewSet(3), NewSet(1)), NewSet(2, 3))
	requireSet(t, BackwardLive(graph, NewSet(5), NewSet(1)), NewSet(4, 5))
	requireSet(t, BackwardLive(graph, NewSet(3, 5), NewSet(0)), NewSet(1, 2, 3, 4, 5))
}

// TestBackwardLiveAgainstIteration compares the port with an independent fixed-point solver.
func TestBackwardLiveAgainstIteration(t *testing.T) {
	random := rand.New(rand.NewPCG(12, 34))
	for trial := range 100 {
		count := 1 + random.IntN(18)
		successors := make([][]int, count)
		for point := range count {
			for range random.IntN(4) {
				successors[point] = append(successors[point], random.IntN(count))
			}
		}
		graph := NewGraph(successors)
		uses, definitions := NewSet(random.IntN(count), random.IntN(count)), NewSet(random.IntN(count))
		expected := referenceLiveness(graph, uses, definitions)
		requireSet(t, BackwardLive(graph, uses, definitions), expected)
	}
}

// referenceLiveness propagates successors backward by repeated full scans rather than predecessor traversal.
func referenceLiveness(graph Graph, uses, definitions Set) Set {
	live := NewSet()
	for changed := true; changed; {
		changed = false
		for point, successors := range graph.Succ {
			if definitions[point] {
				continue
			}
			needed := uses[point]
			for _, successor := range successors {
				needed = needed || live[successor]
			}
			if needed && live.Add(point) {
				changed = true
			}
		}
	}
	return live
}

// TestReversePostorder excludes disconnected nodes and visits loop nodes once.
func TestReversePostorder(t *testing.T) {
	graph := NewGraph([][]int{{1}, {2, 3}, {1}, nil, {0}})
	if got := graph.ReversePostorder(); !slices.Equal(got, []int{0, 1, 3, 2}) {
		t.Fatalf("order=%v", got)
	}
}

// requireSet reports a set mismatch without depending on iteration order.
func requireSet(t *testing.T, got, want Set) {
	t.Helper()
	if !maps.Equal(got, want) {
		t.Fatalf("set=%v want=%v", got, want)
	}
}
