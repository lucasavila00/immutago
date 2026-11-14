package checker

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

// stressGraph builds marked pointer graphs with an optional mutable field.
func stressGraph(edges [][]int, bad int, aliases bool) []*types.Named {
	marker, _ := testTypes()
	nodes := make([]*types.Named, len(edges))
	for i := range nodes {
		nodes[i] = markedType(fmt.Sprintf("Node%d", i), marker)
	}
	for i, targets := range edges {
		fields := []*types.Var{types.NewField(token.NoPos, nil, "Shared", marker, true)}
		for j, target := range targets {
			var element types.Type = nodes[target]
			if aliases {
				element = types.NewAlias(types.NewTypeName(token.NoPos, nil, fmt.Sprintf("Alias%d", j), nil), element)
			}
			fields = append(fields, types.NewField(token.NoPos, nil, fmt.Sprintf("Edge%d", j), types.NewPointer(element), false))
		}
		if i == bad {
			fields = append(fields, types.NewField(token.NoPos, nil, "Mutable", types.NewSlice(types.Typ[types.Int]), false))
		}
		nodes[i].SetUnderlying(types.NewStruct(fields, nil))
	}
	return nodes
}

// stressReaches checks integer edges independently of compiler types and proof state.
func stressReaches(edges [][]int, root, target int) bool {
	seen := make([]bool, len(edges))
	pending := []int{root}
	for len(pending) > 0 {
		node := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if seen[node] {
			continue
		}
		seen[node] = true
		if node == target {
			return true
		}
		pending = append(pending, edges[node]...)
	}
	return false
}

// TestStressProofReachability checks all edge patterns in a small cyclic graph.
func TestStressProofReachability(t *testing.T) {
	for mask := 0; mask < 256; mask++ {
		edges := make([][]int, 4)
		for node := range edges {
			for edge := 0; edge < 2; edge++ {
				if mask&(1<<(node*2+edge)) != 0 {
					edges[node] = append(edges[node], (node+edge+1)%4)
				}
			}
		}
		for bad := -1; bad < len(edges); bad++ {
			nodes := stressGraph(edges, bad, mask%2 == 0)
			for root, node := range nodes {
				reason := Reason(node)
				if got, want := reason != "", stressReaches(edges, root, bad); got != want {
					t.Fatalf("mask=%d bad=%d root=%d: failure=%t, want %t; reason=%q", mask, bad, root, got, want, reason)
				}
				if !Contains(node) {
					t.Fatalf("mask=%d root=%d: missed marked storage", mask, root)
				}
			}
		}
	}
}

// stressEdges builds chains, repeated-edge fanout graphs, and cycles.
func stressEdges(size int, shape string) [][]int {
	edges := make([][]int, size)
	for i := 0; i < size-1; i++ {
		edges[i] = []int{i + 1}
		if shape == "Fanout" {
			edges[i] = append(edges[i], i+1, size-1)
		}
	}
	if shape == "Cycle" {
		edges[size-1] = []int{0}
	}
	return edges
}

// TestStressLargeProofGraphs checks deep failures after aliases and recursive edges.
func TestStressLargeProofGraphs(t *testing.T) {
	for _, shape := range []string{"Chain", "Fanout", "Cycle"} {
		for _, size := range []int{64, 256, 1024} {
			t.Run(fmt.Sprintf("%s/%d", shape, size), func(t *testing.T) {
				edges := stressEdges(size, shape)
				for _, bad := range []int{-1, size - 1} {
					root := stressGraph(edges, bad, true)[0]
					reason := Reason(root)
					if (reason != "") != (bad >= 0) {
						t.Fatalf("bad=%d: reason=%q", bad, reason)
					}
					if bad >= 0 && !strings.Contains(reason, "Mutable") {
						t.Fatalf("lost failing field: %q", reason)
					}
				}
			})
		}
	}
}

// TestStressAliasMarkerPolicy checks long alias chains for both marker policies.
func TestStressAliasMarkerPolicy(t *testing.T) {
	shared, _ := testTypes()
	localName := types.NewTypeName(token.NoPos, shared.Obj().Pkg(), "LocalRef", nil)
	local := types.NewNamed(localName, types.NewStruct(nil, nil), nil)
	for _, marker := range []*types.Named{shared, local} {
		var alias types.Type = marker
		for i := 0; i < 128; i++ {
			alias = types.NewAlias(types.NewTypeName(token.NoPos, nil, fmt.Sprintf("Alias%d", i), nil), alias)
		}
		value := markedType("Value", alias)
		if !Marked(value) || Reason(value) != "" {
			t.Fatalf("marker alias rejected: %s", marker.Obj().Name())
		}
		stored := types.NewArray(types.NewPointer(value), 2)
		if got, want := Reason(stored) != "", marker == local; got != want {
			t.Fatalf("%s pointer rejection=%t, want %t", marker.Obj().Name(), got, want)
		}
		if !Contains(stored) {
			t.Fatal("alias pointer array hides protected storage")
		}
	}
}

// stressContainsGraph builds unmarked cycles with or without a reachable protected leaf.
func stressContainsGraph(size int, protected bool) types.Type {
	nodes := make([]*types.Named, size)
	for i := range nodes {
		name := types.NewTypeName(token.NoPos, nil, fmt.Sprintf("Plain%d", i), nil)
		nodes[i] = types.NewNamed(name, types.NewStruct(nil, nil), nil)
	}
	marker, _ := testTypes()
	for i, node := range nodes {
		fields := []*types.Var{types.NewField(token.NoPos, nil, "Next", types.NewPointer(nodes[(i+1)%size]), false)}
		if protected && i == size-1 {
			fields = append(fields, types.NewField(token.NoPos, nil, "Value", markedType("Value", marker), false))
		}
		node.SetUnderlying(types.NewStruct(fields, nil))
	}
	return nodes[0]
}

// TestStressContainsCycles checks detection after a long unmarked cycle.
func TestStressContainsCycles(t *testing.T) {
	for _, size := range []int{64, 256, 1024} {
		for _, protected := range []bool{false, true} {
			if got := Contains(stressContainsGraph(size, protected)); got != protected {
				t.Fatalf("size=%d: contains=%t, want %t", size, got, protected)
			}
		}
	}
}
