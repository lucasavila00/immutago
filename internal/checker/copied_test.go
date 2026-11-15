package checker

import (
	"go/token"
	"go/types"
	"slices"
	"testing"
)

// TestCopiedRecordCategories checks recursive value proofs and mutable holder classification.
func TestCopiedRecordCategories(t *testing.T) {
	frozen, _ := testTypes()
	marker := types.NewNamed(types.NewTypeName(token.NoPos, frozen.Obj().Pkg(), "Copied", nil),
		types.NewStruct(nil, nil), nil)
	score := markedType("Score", marker, types.NewField(token.NoPos, nil, "N", types.Typ[types.Int], false))
	pair := markedType("Pair", marker, types.NewField(token.NoPos, nil, "Scores", types.NewArray(score, 2), false))
	alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Alias", nil), pair)
	holder := types.NewStruct([]*types.Var{types.NewField(token.NoPos, nil, "Score", score, false)}, nil)
	shared := markedType("Shared", frozen, types.NewField(token.NoPos, nil, "Score", score, false))
	bad := markedType("Bad", marker, types.NewField(token.NoPos, nil, "Text", types.Typ[types.String], false))
	badArray := markedType("BadArray", marker, types.NewField(token.NoPos, nil, "Values", types.NewArray(bad, 0), false))
	cases := []struct {
		typ                       types.Type
		frozen, copied, protected bool
	}{
		{score, true, true, false},
		{pair, true, true, false},
		{alias, true, true, false},
		{types.NewArray(pair, 0), true, true, false},
		{holder, false, false, false},
		{types.NewPointer(holder), false, false, false},
		{types.NewArray(holder, 2), false, false, false},
		{types.NewMap(types.Typ[types.String], holder), false, false, false},
		{types.NewPointer(score), false, false, false},
		{types.NewSlice(score), false, false, false},
		{shared, true, false, true},
		{types.NewArray(shared, 2), true, false, true},
		{bad, false, false, false},
		{badArray, false, false, false},
	}
	for range 2 {
		var proofs Prover
		var queries Classifier
		for _, test := range cases {
			if got := proofs.FrozenReason(test.typ) == ""; got != test.frozen {
				t.Errorf("Frozen(%s)=%v; want %v", test.typ, got, test.frozen)
			}
			if got := proofs.CopiedReason(test.typ) == ""; got != test.copied {
				t.Errorf("Copied(%s)=%v; want %v", test.typ, got, test.copied)
			}
			if got := proofs.ReadResultReason(test.typ) == ""; got != test.copied {
				t.Errorf("ReadResult(%s)=%v; want %v", test.typ, got, test.copied)
			}
			if got := queries.Contains(test.typ); got != test.protected {
				t.Errorf("Contains(%s)=%v; want %v", test.typ, got, test.protected)
			}
		}
		slices.Reverse(cases)
	}
}
