package checker

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

// TestProverReachability compares reused proofs with independent reachability in both query orders.
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

// checkProverGraph checks rejection and real cause paths against the generated type graph.
func checkProverGraph(t *testing.T, nodes []*types.Named, edges [][]int, bad int, reverse bool) {
	t.Helper()
	var prover Prover
	for index := range nodes {
		root := index
		if reverse {
			root = len(nodes) - 1 - index
		}
		reason := prover.Reason(nodes[root])
		if failed := stressReaches(edges, root, bad); (reason != "") != failed {
			t.Fatalf("edges=%v bad=%d root=%d: reason=%q, reachable=%t", edges, bad, root, reason, failed)
		}
		if reason != "" {
			checkProverWitness(t, &prover, nodes[root])
			if !strings.HasSuffix(reason, "Mutable: []int is not provably immutable") {
				t.Fatalf("lost original mutable field: %q", reason)
			}
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

// competingProofRoots creates a cycle with two failure sources and a repeated edge.
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

// TestProverWitnessQueryOrder checks that every root stays invalid and has a real acyclic cause path.
func TestProverWitnessQueryOrder(t *testing.T) {
	roots := competingProofRoots()
	for first := range roots {
		var prover Prover
		for offset := range roots {
			root := roots[(first+offset)%len(roots)]
			reason := prover.Reason(root)
			if !strings.Contains(reason, "BadA: []int") && !strings.Contains(reason, "BadB: map[int]int") {
				t.Fatalf("first=%d root=%s: missing original failure in %q", first, root, reason)
			}
			checkProverWitness(t, &prover, root)
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
		reason := prover.Reason(root)
		if !strings.Contains(reason, "Mutable: []int") {
			t.Fatalf("root=%s: missing failure in %q", root, reason)
		}
		checkProverWitness(t, &prover, root)
	}
}

// checkProverWitness verifies each cause edge against Go type components and rejects cause cycles.
func checkProverWitness(t *testing.T, prover *Prover, root types.Type) {
	t.Helper()
	nodeTypes := make(map[*obligation]types.Type)
	for typ, node := range prover.graph.nodes {
		nodeTypes[node] = typ
	}
	seen := make(map[*obligation]bool)
	node := prover.graph.nodes[types.Unalias(root)]
	for node.cause != nil {
		if seen[node] {
			t.Fatal("failure witness contains a cycle")
		}
		seen[node] = true
		if node.cause.reason == "" {
			t.Fatal("cause points to empty text")
		}
		if !validWitnessEdge(nodeTypes[node], nodeTypes[node.cause], node.reason) {
			t.Fatalf("invalid witness edge from %s to %s with label %q", nodeTypes[node], nodeTypes[node.cause], node.reason)
		}
		node = node.cause
	}
	if node.reason == "" {
		t.Fatal("failure witness has no terminal reason")
	}
}

// validWitnessEdge uses compiler type identity to check the actual field, element, or type argument.
func validWitnessEdge(parent, child types.Type, label string) bool {
	if IsCollection(parent) {
		args := parent.(*types.Named).TypeArgs()
		for i := 0; i < args.Len(); i++ {
			if label == fmt.Sprintf("type argument %d: ", i+1) && witnessEmptyPath(args.At(i), child) {
				return true
			}
		}
		return false
	}
	switch value := parent.Underlying().(type) {
	case *types.Pointer:
		return label == "" && witnessEmptyPath(value.Elem(), child)
	case *types.Array:
		return label == "" && witnessEmptyPath(value.Elem(), child)
	case *types.Struct:
		for field := range value.Fields() {
			if label == field.Name()+": " && witnessEmptyPath(field.Type(), child) {
				return true
			}
		}
	}
	return false
}

// witnessEmptyPath follows only pointer and unnamed-array steps that have no diagnostic label.
func witnessEmptyPath(parent, child types.Type) bool {
	seen := make(map[types.Type]bool)
	for !seen[parent] {
		if types.Identical(parent, child) {
			return true
		}
		seen[parent] = true
		switch value := parent.Underlying().(type) {
		case *types.Pointer:
			parent = value.Elem()
		case *types.Array:
			if _, named := types.Unalias(parent).(*types.Named); named {
				return false
			}
			parent = value.Elem()
		default:
			return false
		}
	}
	return false
}

// TestProverEmptyPaths bounds cause traversal on many roots of one shared array chain.
func TestProverEmptyPaths(t *testing.T) {
	var roots []types.Type
	var typ types.Type = types.NewSlice(types.Typ[types.Int])
	for i := 0; i < 256; i++ {
		typ = types.NewArray(typ, 1)
		roots = append(roots, typ)
	}
	for _, reverse := range []bool{false, true} {
		var prover Prover
		for index := range roots {
			query := index
			if reverse {
				query = len(roots) - 1 - index
			}
			typ := roots[query]
			if reason := prover.Reason(typ); reason != "[]int is not provably immutable" {
				t.Fatalf("array depth=%d: reason=%q", query, reason)
			}
			root := prover.graph.nodes[typ]
			if root.cause == nil || root.cause.cause != nil || root.cause.reason == "" {
				t.Fatal("empty array edges were not collapsed to one visible failure")
			}
			checkProverWitness(t, &prover, typ)
		}
	}
}

// TestProverLocalFailures keeps a direct field failure from acquiring an unnecessary Next path.
func TestProverLocalFailures(t *testing.T) {
	marker, _ := testTypes()
	var prover Prover
	var child types.Type = markedType("Leaf", marker,
		types.NewField(token.NoPos, nil, "Bad", types.NewSlice(types.Typ[types.Int]), false))
	_ = prover.Reason(child)
	for i := 0; i < 64; i++ {
		child = markedType("Parent", marker,
			types.NewField(token.NoPos, nil, "Next", types.NewPointer(child), false),
			types.NewField(token.NoPos, nil, "Bad", types.NewSlice(types.Typ[types.Int]), false))
		if reason := prover.Reason(child); reason != "Bad: []int is not provably immutable" {
			t.Fatalf("unnecessary indirect path at depth %d: %q", i, reason)
		}
		checkProverWitness(t, &prover, child)
	}
}

// TestProverTypeForms checks valid forms and useful terminal reasons without choosing between failures.
func TestProverTypeForms(t *testing.T) {
	marker, slice := testTypes()
	shared := markedType("SharedValue", marker)
	scopedMarker := types.NewNamed(types.NewTypeName(token.NoPos, marker.Obj().Pkg(), "Scoped", nil),
		types.NewStruct(nil, nil), nil)
	scoped := markedType("ScopedValue", scopedMarker)
	alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Alias", nil), shared)
	owner := types.NewNamed(types.NewTypeName(token.NoPos, marker.Obj().Pkg(), "Owner", nil),
		types.NewStruct(nil, nil), nil)
	parameter := types.NewTypeParam(types.NewTypeName(token.NoPos, nil, "T", nil),
		types.NewInterfaceType(nil, nil).Complete())
	shapes := []struct {
		typ     types.Type
		failure string
	}{
		{types.Typ[types.Int], ""}, {shared, ""}, {scoped, ""}, {alias, ""},
		{types.NewPointer(shared), ""}, {types.NewArray(alias, 0), ""}, {sliceType(t, slice, shared), ""},
		{types.NewPointer(scoped), "Scoped pointers cannot be stored"},
		{owner, "Owner slots are mutable"}, {parameter, "type parameters are unsupported"},
		{types.NewPointer(types.Typ[types.Int]), "concrete marked struct target"},
		{types.NewStruct(nil, nil), "struct does not embed"},
	}
	for _, reverse := range []bool{false, true} {
		var prover Prover
		for index := range shapes {
			query := index
			if reverse {
				query = len(shapes) - 1 - index
			}
			shape := shapes[query]
			reason := prover.Reason(shape.typ)
			if (reason != "") != (shape.failure != "") || !strings.Contains(reason, shape.failure) {
				t.Fatalf("shape=%s reverse=%t: reason=%q, want failure %q", shape.typ, reverse, reason, shape.failure)
			}
			if reason != "" {
				checkProverWitness(t, &prover, shape.typ)
			}
		}
	}
}
