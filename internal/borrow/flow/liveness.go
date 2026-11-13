// Copyright (c) The Rust Project Contributors
// Adapted from rustc_borrowck/src/type_check/liveness/trace.rs and dataflow.rs.
// Source revision and license: third_party/rustc.

package flow

// BackwardLive traces each use backwards to its reaching definitions, as rustc's
// compute_use_live_points_for does. Uses are included; definitions are excluded.
// The Go lowering separates use and definition points to preserve evaluation order.
func BackwardLive(graph Graph, uses, definitions Set) Set {
	live := NewSet()
	stack := make([]int, 0, len(uses))
	for point := range uses {
		stack = append(stack, point)
	}
	for len(stack) > 0 {
		point := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if definitions[point] || !live.Add(point) {
			continue
		}
		stack = append(stack, graph.Pred[point]...)
	}
	return live
}

// OutOfScope finds the first points outside a region on paths from issuance.
// It translates OutOfScopePrecomputer's walk over normalized operation points.
// A live issuance point must be included in region by the caller.
func OutOfScope(graph Graph, issuance int, region Set) Set {
	kills, visited := NewSet(), NewSet()
	stack := []int{issuance}
	for len(stack) > 0 {
		point := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if !visited.Add(point) {
			continue
		}
		if !region[point] {
			kills.Add(point)
			continue
		}
		stack = append(stack, graph.Succ[point]...)
	}
	return kills
}
