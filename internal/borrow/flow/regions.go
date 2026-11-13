// Copyright (c) The Rust Project Contributors
// Adapted from rustc_borrowck/src/region_infer/region_context.rs::propagate_constraints.
// Source revision and license: third_party/rustc.

package flow

// SolveRegions propagates liveness through outlives constraints after collapsing cycles.
// An edge [sup, sub] requires sup to contain every point in sub. Input sets are not changed.
// SCC values are solved in dependency order, matching rustc's region propagation.
func SolveRegions(seeds []Set, outlives [][2]int) []Set {
	successors := make([][]int, len(seeds))
	for _, edge := range outlives {
		successors[edge[0]] = append(successors[edge[0]], edge[1])
	}
	components, membership := stronglyConnected(successors)
	values := make([]Set, len(components))
	for component, nodes := range components {
		value := NewSet()
		for _, node := range nodes {
			value.Union(seeds[node])
		}
		for _, node := range nodes {
			for _, child := range successors[node] {
				dependency := membership[child]
				if dependency != component {
					value.Union(values[dependency])
				}
			}
		}
		values[component] = value
	}
	result := make([]Set, len(seeds))
	for node, component := range membership {
		result[node] = values[component].Clone()
	}
	return result
}

// sccSearch maintains Tarjan's active path while creating SCCs in dependency order.
type sccSearch struct {
	successors [][]int
	index      []int
	low        []int
	active     []bool
	stack      []int
	next       int
	components [][]int
	membership []int
}

// stronglyConnected returns components in dependency order and the component of each node.
func stronglyConnected(successors [][]int) ([][]int, []int) {
	count := len(successors)
	search := &sccSearch{
		successors: successors, index: make([]int, count), low: make([]int, count),
		active: make([]bool, count), membership: make([]int, count),
	}
	for node := range count {
		search.index[node] = -1
	}
	for node := range count {
		if search.index[node] == -1 {
			search.visit(node)
		}
	}
	return search.components, search.membership
}

// visit finishes every successor component before emitting its parent component.
func (s *sccSearch) visit(node int) {
	s.index[node], s.low[node] = s.next, s.next
	s.next++
	s.stack = append(s.stack, node)
	s.active[node] = true
	for _, child := range s.successors[node] {
		if s.index[child] == -1 {
			s.visit(child)
			s.low[node] = min(s.low[node], s.low[child])
		} else if s.active[child] {
			s.low[node] = min(s.low[node], s.index[child])
		}
	}
	if s.low[node] != s.index[node] {
		return
	}
	var component []int
	for {
		member := s.stack[len(s.stack)-1]
		s.stack = s.stack[:len(s.stack)-1]
		s.active[member] = false
		s.membership[member] = len(s.components)
		component = append(component, member)
		if member == node {
			break
		}
	}
	s.components = append(s.components, component)
}
