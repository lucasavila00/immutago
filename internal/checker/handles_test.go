package checker

import (
	"go/token"
	"go/types"
	"testing"
)

// TestHandleShape checks the public shape without using a runtime type name.
func TestHandleShape(t *testing.T) {
	marker, _ := testTypes()
	unchecked := types.NewNamed(types.NewTypeName(token.NoPos, marker.Obj().Pkg(), "UncheckedValue", nil),
		types.NewStruct([]*types.Var{types.NewField(token.NoPos, nil, "value", types.NewPointer(types.Typ[types.Int]), false)}, nil), nil)
	payload := types.NewField(token.NoPos, nil, "payload", unchecked, false)
	handle := markedType("Handle", marker, payload)
	alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Alias", nil), handle)
	extra := markedType("Extra", marker, payload, types.NewField(token.NoPos, nil, "N", types.Typ[types.Int], false))
	raw := markedType("Raw", marker, types.NewField(token.NoPos, nil, "value", types.NewPointer(types.Typ[types.Int]), false))
	for _, test := range []struct {
		typ    types.Type
		handle bool
	}{
		{handle, true}, {alias, true}, {extra, false}, {raw, false}, {unchecked, false}, {types.NewPointer(handle), false},
	} {
		if IsHandle(test.typ) != test.handle {
			t.Errorf("IsHandle(%s) = %v", test.typ, !test.handle)
		}
	}
	for _, typ := range []types.Type{handle, alias, types.NewArray(handle, 2)} {
		var proofs Prover
		if proofs.FrozenReason(typ) != "" || proofs.ReadResultReason(typ) != "" || proofs.CopiedReason(typ) == "" {
			t.Errorf("handle categories disagree for %s", typ)
		}
		var queries Classifier
		if !queries.Contains(typ) || queries.Owns(typ) || queries.Transport(typ) || queries.Callable(typ) {
			t.Errorf("handle classification disagrees for %s", typ)
		}
	}
}
