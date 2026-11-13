// Copyright (c) The Rust Project Contributors
// Adapted from rustc_mir_dataflow/src/framework/mod.rs::iterate_to_fixpoint.
// Source revision and license: third_party/rustc.

package flow

import "slices"

// Graph records normalized operation points. Point zero is the entry.
// Pred and Succ must describe the same directed edges.
type Graph struct {
	Succ [][]int
	Pred [][]int
}

// NewGraph constructs predecessor edges and copies the supplied successor lists.
func NewGraph(successors [][]int) Graph {
	graph := Graph{Succ: make([][]int, len(successors)), Pred: make([][]int, len(successors))}
	for from, targets := range successors {
		graph.Succ[from] = slices.Clone(targets)
		for _, to := range targets {
			graph.Pred[to] = append(graph.Pred[to], from)
		}
	}
	return graph
}

// ReversePostorder returns reachable points in forward dataflow order.
func (g Graph) ReversePostorder() []int {
	if len(g.Succ) == 0 {
		return nil
	}
	visited := make([]bool, len(g.Succ))
	var order []int
	var visit func(int)
	visit = func(point int) {
		if visited[point] {
			return
		}
		visited[point] = true
		for _, successor := range g.Succ[point] {
			visit(successor)
		}
		order = append(order, point)
	}
	visit(0)
	slices.Reverse(order)
	return order
}

// Forward solves a union gen/kill analysis using rustc's minimum-rank dirty worklist.
// Inputs are states before the point's kills; outputs are states after kills and generation.
// Callers must check accesses before applying assignment kills. Unreachable points remain empty.
func Forward(graph Graph, gen, kill []Set) (inputs, outputs []Set) {
	count := len(graph.Succ)
	inputs, outputs = make([]Set, count), make([]Set, count)
	for point := range count {
		inputs[point], outputs[point] = NewSet(), NewSet()
	}
	order := graph.ReversePostorder()
	ranks, dirty := make([]int, count), make([]bool, len(order))
	for rank, point := range order {
		ranks[point], dirty[rank] = rank, true
	}
	for current := 0; current < len(order); {
		point := order[current]
		dirty[current] = false
		state := inputs[point].Without(kill[point])
		state.Union(gen[point])
		outputs[point] = state
		for _, target := range graph.Succ[point] {
			if inputs[target].Union(state) {
				rank := ranks[target]
				dirty[rank] = true
				current = min(current, rank)
			}
		}
		for current < len(order) && !dirty[current] {
			current++
		}
	}
	return inputs, outputs
}
