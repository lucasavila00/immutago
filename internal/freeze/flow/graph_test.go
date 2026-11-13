package flow

import (
	"slices"
	"testing"
)

// TestReversePostorder excludes disconnected nodes and visits loop nodes once.
func TestReversePostorder(t *testing.T) {
	graph := NewGraph([][]int{{1}, {2, 3}, {1}, nil, {0}})
	if got := graph.ReversePostorder(); !slices.Equal(got, []int{0, 1, 3, 2}) {
		t.Fatalf("order=%v", got)
	}
}
