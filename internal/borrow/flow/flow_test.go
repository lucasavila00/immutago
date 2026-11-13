package flow

import (
	"maps"
	"math/rand/v2"
	"testing"
)

// TestBackwardLive traces branch and loop uses to definitions, not textual scope ends.
func TestBackwardLive(t *testing.T) {
	graph := NewGraph([][]int{{1}, {2, 4}, {3}, {1}, {5}, nil})
	live := BackwardLive(graph, NewSet(3), NewSet(1))
	requireSet(t, live, NewSet(2, 3))
	live = BackwardLive(graph, NewSet(5), NewSet(1))
	requireSet(t, live, NewSet(4, 5))
	live = BackwardLive(graph, NewSet(3, 5), NewSet(0))
	requireSet(t, live, NewSet(1, 2, 3, 4, 5))
}

// TestRegionPropagation checks outlives direction and cycle collapse without mutating seeds.
func TestRegionPropagation(t *testing.T) {
	seeds := []Set{NewSet(0), NewSet(1), NewSet(2), NewSet(3)}
	result := SolveRegions(seeds, [][2]int{{0, 1}, {1, 2}, {2, 1}})
	requireSet(t, result[0], NewSet(0, 1, 2))
	requireSet(t, result[1], NewSet(1, 2))
	requireSet(t, result[2], NewSet(1, 2))
	requireSet(t, result[3], NewSet(3))
	result[1].Add(99)
	requireSet(t, result[2], NewSet(1, 2))
	requireSet(t, seeds[0], NewSet(0))
}

// TestOutOfScope stops each path at its first dead point even if the region later contains points.
func TestOutOfScope(t *testing.T) {
	graph := NewGraph([][]int{{1, 3}, {2}, {4}, {4}, {1}})
	requireSet(t, OutOfScope(graph, 0, NewSet(0, 1, 2, 4)), NewSet(3))
	requireSet(t, OutOfScope(graph, 0, NewSet(0, 1, 4)), NewSet(2, 3))
}

// TestForwardKeepsPreKillEvidence verifies that a conflicting write cannot erase its incoming loan.
func TestForwardKeepsPreKillEvidence(t *testing.T) {
	graph := NewGraph([][]int{{1}, {2}, {1, 3}, nil, {3}})
	gen := []Set{NewSet(7), nil, NewSet(8), nil, NewSet(99)}
	kill := []Set{nil, NewSet(7), nil, nil, nil}
	inputs, outputs := Forward(graph, gen, kill)
	requireSet(t, inputs[1], NewSet(7, 8))
	requireSet(t, outputs[1], NewSet(8))
	requireSet(t, inputs[3], NewSet(8))
	requireSet(t, outputs[4], NewSet())
}

// TestRegionSolverAgainstClosure compares SCC propagation with an independent monotone reference solver.
func TestRegionSolverAgainstClosure(t *testing.T) {
	random := rand.New(rand.NewPCG(12, 34))
	for trial := range 100 {
		count := 1 + random.IntN(18)
		seeds := make([]Set, count)
		for i := range count {
			seeds[i] = NewSet(i)
		}
		var edges [][2]int
		for range count * 3 {
			edges = append(edges, [2]int{random.IntN(count), random.IntN(count)})
		}
		expected := cloneSets(seeds)
		for changed := true; changed; {
			changed = false
			for _, edge := range edges {
				if expected[edge[0]].Union(expected[edge[1]]) {
					changed = true
				}
			}
		}
		if got := SolveRegions(seeds, edges); !equalSets(got, expected) {
			t.Fatalf("trial %d: regions=%v want=%v", trial, got, expected)
		}
	}
}

// TestForwardAgainstIteration checks the ranked worklist on irregular CFGs with loops and unreachable nodes.
func TestForwardAgainstIteration(t *testing.T) {
	random := rand.New(rand.NewPCG(56, 78))
	for trial := range 100 {
		count := 1 + random.IntN(15)
		successors := make([][]int, count)
		gen, kill := make([]Set, count), make([]Set, count)
		for point := range count {
			for range random.IntN(4) {
				successors[point] = append(successors[point], random.IntN(count))
			}
			gen[point], kill[point] = NewSet(random.IntN(6)), NewSet(random.IntN(6))
		}
		graph := NewGraph(successors)
		expectedIn, expectedOut := referenceForward(graph, gen, kill)
		gotIn, gotOut := Forward(graph, gen, kill)
		if !equalSets(gotIn, expectedIn) || !equalSets(gotOut, expectedOut) {
			t.Fatalf("trial %d: worklist differs from full iteration", trial)
		}
	}
}

// referenceForward is a deliberately unranked repeated scan used as an independent test oracle.
func referenceForward(graph Graph, gen, kill []Set) (inputs, outputs []Set) {
	count := len(graph.Succ)
	inputs, outputs = make([]Set, count), make([]Set, count)
	for point := range count {
		inputs[point], outputs[point] = NewSet(), NewSet()
	}
	reachable := NewSet(graph.ReversePostorder()...)
	for changed := true; changed; {
		changed = false
		for point := range count {
			if !reachable[point] {
				continue
			}
			out := inputs[point].Without(kill[point])
			out.Union(gen[point])
			outputs[point] = out
			for _, target := range graph.Succ[point] {
				if inputs[target].Union(out) {
					changed = true
				}
			}
		}
	}
	return inputs, outputs
}

// cloneSets copies a list without sharing mutable set storage.
func cloneSets(sets []Set) []Set {
	result := make([]Set, len(sets))
	for index, set := range sets {
		result[index] = set.Clone()
	}
	return result
}

// requireSet reports a set mismatch without depending on iteration order.
func requireSet(t *testing.T, got, want Set) {
	t.Helper()
	if !maps.Equal(got, want) {
		t.Fatalf("set=%v want=%v", got, want)
	}
}

// equalSets compares region lists independently of set iteration order.
func equalSets(left, right []Set) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !maps.Equal(left[index], right[index]) {
			return false
		}
	}
	return true
}
