package checker

import (
	"go/token"
	"go/types"
	"testing"
)

// TestProverReachability compares reused proofs with independent reachability and fresh diagnostic order.
func TestProverReachability(t *testing.T) {
	for mask := 0; mask < 256; mask++ {
		edges := stressPatternEdges(mask)
		for bad := -1; bad < len(edges); bad++ {
			nodes := stressGraph(edges, bad, mask%2 == 0)
			for _, reverse := range []bool{false, true} {
				checkProverGraph(t, nodes, edges, bad, reverse)
			}
		}
	}
}

// checkProverGraph changes query order without sharing the original walker's state.
func checkProverGraph(t *testing.T, nodes []*types.Named, edges [][]int, bad int, reverse bool) {
	t.Helper()
	var prover Prover
	for index := range nodes {
		root := index
		if reverse {
			root = len(nodes) - 1 - index
		}
		got, want := prover.Reason(nodes[root]), originalReason(nodes[root])
		if got != want {
			t.Fatalf("edges=%v bad=%d root=%d reverse=%t: got %q, want %q", edges, bad, root, reverse, got, want)
		}
		if failed := stressReaches(edges, root, bad); (got != "") != failed {
			t.Fatalf("edges=%v bad=%d root=%d: reason=%q, reachable=%t", edges, bad, root, got, failed)
		}
	}
	count := len(prover.graph.nodes)
	for _, node := range nodes {
		_ = prover.Reason(node)
	}
	if len(prover.graph.nodes) != count {
		t.Fatal("repeated queries rebuilt proof nodes")
	}
}

// competingProofRoots creates a cycle where the shared discovery order would change a later root's message.
func competingProofRoots() []types.Type {
	marker, _ := testTypes()
	root, left, right := markedType("Root", marker), markedType("Left", marker), markedType("Right", marker)
	root.SetUnderlying(types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "Shared", marker, true),
		types.NewField(token.NoPos, nil, "Left", types.NewPointer(left), false),
		types.NewField(token.NoPos, nil, "Again", types.NewPointer(left), false),
		types.NewField(token.NoPos, nil, "Right", types.NewPointer(right), false),
	}, nil))
	left.SetUnderlying(types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "Shared", marker, true),
		types.NewField(token.NoPos, nil, "Back", types.NewPointer(root), false),
		types.NewField(token.NoPos, nil, "BadA", types.NewSlice(types.Typ[types.Int]), false),
	}, nil))
	right.SetUnderlying(types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "Shared", marker, true),
		types.NewField(token.NoPos, nil, "BadB", types.NewMap(types.Typ[types.Int], types.Typ[types.Int]), false),
	}, nil))
	alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Alias", nil), root)
	return []types.Type{root, left, right, alias}
}

// TestProverDiagnosticQueryOrder preserves exact messages across cycles, repeated edges, aliases, and ties.
func TestProverDiagnosticQueryOrder(t *testing.T) {
	roots := competingProofRoots()
	for first := range roots {
		var prover Prover
		for offset := range roots {
			root := roots[(first+offset)%len(roots)]
			if got, want := prover.Reason(root), originalReason(root); got != want {
				t.Fatalf("first=%d root=%s: got %q, want %q", first, root, got, want)
			}
		}
		if got, want := prover.Reason(roots[0]), "Left: BadA: []int is not provably immutable"; got != want {
			t.Fatalf("first=%d: got %q, want %q", first, got, want)
		}
	}
}

// TestProverNewFailedOwners checks new edges to an already failed child, including empty pointer labels.
func TestProverNewFailedOwners(t *testing.T) {
	marker, slice := testTypes()
	bad := markedType("Bad", marker,
		types.NewField(token.NoPos, nil, "Mutable", types.NewSlice(types.Typ[types.Int]), false))
	roots := []types.Type{bad, types.NewPointer(bad), sliceType(t, slice, bad)}
	roots = append(roots, markedType("Parent", marker,
		types.NewField(token.NoPos, nil, "Value", roots[2], false)))
	var prover Prover
	for _, root := range roots {
		if got, want := prover.Reason(root), originalReason(root); got == "" || got != want {
			t.Fatalf("root=%s: got %q, want %q", root, got, want)
		}
	}
}

// TestProverTypeForms preserves terminal failures and valid proofs for all existing type shapes.
func TestProverTypeForms(t *testing.T) {
	shapes := classificationShapes(t)[1:]
	marker, _ := testTypes()
	owner := types.NewNamed(types.NewTypeName(token.NoPos, marker.Obj().Pkg(), "Owner", nil),
		types.NewStruct(nil, nil), nil)
	shapes = append(shapes, owner, types.NewPointer(types.Typ[types.Int]))
	for _, reverse := range []bool{false, true} {
		var prover Prover
		for index := range shapes {
			query := index
			if reverse {
				query = len(shapes) - 1 - index
			}
			if got, want := prover.Reason(shapes[query]), originalReason(shapes[query]); got != want {
				t.Fatalf("shape=%s reverse=%t: got %q, want %q", shapes[query], reverse, got, want)
			}
		}
	}
}
