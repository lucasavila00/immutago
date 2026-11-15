package checker

import (
	"go/token"
	"go/types"
	"strings"
	"testing"
)

// TestLocalViewClassification checks stored views and references through aliases and cycles.
func TestLocalViewClassification(t *testing.T) {
	shared, slice := testTypes()
	pkg := shared.Obj().Pkg()
	local := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "FrozenByValueString", nil),
		types.NewStruct([]*types.Var{types.NewField(token.NoPos, pkg, "value", types.Typ[types.String], false)}, nil), nil)
	alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Text", nil), local)
	frozen := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "FrozenString", nil), local.Underlying(), nil)
	var queries Classifier
	cases := []struct {
		typ            types.Type
		views, retains bool
	}{
		{types.Typ[types.Int], false, false},
		{types.Typ[types.String], false, true},
		{frozen, false, true},
		{local, true, false},
		{alias, true, false},
		{types.NewArray(alias, 0), true, false},
		{types.NewSlice(alias), true, false},
		{types.NewMap(types.Typ[types.Int], alias), true, false},
		{types.NewPointer(alias), true, true},
		{sliceType(t, slice, types.Typ[types.Int]), false, true},
		{markedType("Plain", shared, types.NewField(token.NoPos, nil, "N", types.Typ[types.Int], false)), false, false},
	}
	for _, test := range cases {
		views, retains := queries.Views(test.typ), queries.Retains(test.typ)
		if views != test.views || retains != test.retains {
			t.Errorf("%s: Views=%v Retains=%v; want %v %v", test.typ, views, retains, test.views, test.retains)
		}
	}
	for _, order := range [][]int{{0, 1}, {1, 0}} {
		first := types.NewNamed(types.NewTypeName(token.NoPos, nil, "First", nil), types.NewStruct(nil, nil), nil)
		second := types.NewNamed(types.NewTypeName(token.NoPos, nil, "Second", nil), types.NewStruct(nil, nil), nil)
		first.SetUnderlying(types.NewStruct([]*types.Var{
			types.NewField(token.NoPos, nil, "Next", types.NewPointer(second), false),
		}, nil))
		second.SetUnderlying(types.NewStruct([]*types.Var{
			types.NewField(token.NoPos, nil, "Next", types.NewPointer(first), false),
			types.NewField(token.NoPos, nil, "Text", local, false),
		}, nil))
		var cyclic Classifier
		nodes := []types.Type{first, second}
		for _, index := range order {
			if !cyclic.Views(nodes[index]) || !cyclic.Retains(nodes[index]) {
				t.Fatal("recursive field graph lost a view or pointer")
			}
		}
	}
}

// TestLocalViewProof checks recursive field restrictions and the separate result proof.
func TestLocalViewProof(t *testing.T) {
	shared, slice := testTypes()
	pkg := shared.Obj().Pkg()
	byValue := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "ByValue", nil), types.NewStruct(nil, nil), nil)
	local := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "FrozenByValueString", nil),
		types.NewStruct(nil, nil), nil)
	field := types.NewField(token.NoPos, nil, "Text", local, false)
	result := markedType("Result", byValue, field)
	var prover Prover
	for _, typ := range []types.Type{local, result, types.NewArray(result, 2)} {
		if reason := prover.Reason(typ); reason != "" {
			t.Fatalf("local immutable value %s: %s", typ, reason)
		}
		if reason := prover.ServiceResultReason(typ); reason != "" {
			t.Fatalf("local service result %s: %s", typ, reason)
		}
	}
	for _, typ := range []types.Type{
		markedType("SharedResult", shared, field), sliceType(t, slice, result),
	} {
		if reason := prover.Reason(typ); !strings.Contains(reason, "local frozen views") {
			t.Fatalf("persistent storage %s: %q", typ, reason)
		}
	}
	internal := markedType("Internal", byValue,
		types.NewField(token.NoPos, nil, "Items", sliceType(t, slice, types.Typ[types.Int]), false))
	if reason := prover.Reason(internal); reason != "" {
		t.Fatal(reason)
	}
	if reason := prover.ServiceResultReason(internal); !strings.Contains(reason, "shared data references") {
		t.Fatalf("retained data accepted as a service result: %q", reason)
	}
}
