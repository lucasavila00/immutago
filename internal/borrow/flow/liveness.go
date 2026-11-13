// Copyright (c) The Rust Project Contributors
// Adapted from rustc_borrowck/src/type_check/liveness/trace.rs.
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
