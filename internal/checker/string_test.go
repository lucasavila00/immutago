package checker

import (
	"go/token"
	"go/types"
	"strings"
	"testing"
)

// TestFrozenStringProof checks exact identity and recursive native string rejection.
func TestFrozenStringProof(t *testing.T) {
	marker, slice := testTypes()
	field := types.NewField(token.NoPos, marker.Obj().Pkg(), "value", types.Typ[types.String], false)
	frozen := types.NewNamed(types.NewTypeName(token.NoPos, marker.Obj().Pkg(), "FrozenString", nil),
		types.NewStruct([]*types.Var{field}, nil), nil)
	alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Text", nil), frozen)
	for _, typ := range []types.Type{frozen, alias, types.NewArray(frozen, 2), sliceType(t, slice, frozen)} {
		if reason := Reason(typ); reason != "" || !Contains(typ) {
			t.Fatalf("frozen string type %s: reason %q, protected %v", typ, reason, Contains(typ))
		}
	}
	var prover Prover
	if reason := prover.ScopedResultReason(alias); reason != "" {
		t.Fatal(reason)
	}
	native := types.NewNamed(types.NewTypeName(token.NoPos, nil, "Native", nil), types.Typ[types.String], nil)
	nativeAlias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Alias", nil), types.Typ[types.String])
	for _, typ := range []types.Type{
		types.Typ[types.String], native, nativeAlias, types.NewArray(nativeAlias, 0),
		sliceType(t, slice, native), markedType("Value", marker, types.NewField(token.NoPos, nil, "Text", native, false)),
	} {
		if reason := prover.Reason(typ); !strings.Contains(reason, "native strings require im.FrozenString") {
			t.Fatalf("native string type %s: %q", typ, reason)
		}
	}
	other := types.NewNamed(types.NewTypeName(token.NoPos, types.NewPackage("other", "im"), "FrozenString", nil),
		types.NewStruct(nil, nil), nil)
	if IsCollection(other) || Reason(other) == "" {
		t.Fatal("accepted FrozenString from another package")
	}
}
