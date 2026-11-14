// Copyright (c) The Rust Project Contributors
// Uses the reverse-postorder traversal used by rustc_mir_dataflow.
// Source revision and license: third_party/rustc.

// Package flow supplies control-flow graphs for builder consumption checks.
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
