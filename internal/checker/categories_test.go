package checker

import (
	"go/token"
	"go/types"
	"testing"
)

// TestSnapshotCategories proves the same type in each category and query order.
func TestSnapshotCategories(t *testing.T) {
	snapshot, _ := testTypes()
	pkg := snapshot.Obj().Pkg()
	local := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Local", nil), types.NewStruct(nil, nil), nil)
	parameter := types.NewTypeParam(types.NewTypeName(token.NoPos, pkg, "T", nil),
		types.NewInterfaceType(nil, nil).Complete())
	projection := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Projection", nil),
		types.NewStruct([]*types.Var{types.NewField(token.NoPos, pkg, "value", parameter, false)}, nil), nil)
	projection.SetTypeParams([]*types.TypeParam{parameter})
	text := sliceType(t, projection, types.Typ[types.String])
	frozenText := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "FrozenString", nil),
		types.NewStruct([]*types.Var{types.NewField(token.NoPos, pkg, "value", types.Typ[types.String], false)}, nil), nil)
	copied := markedType("Count", snapshot, types.NewField(token.NoPos, nil, "N", types.Typ[types.Int], false))
	retained := markedType("Text", snapshot, types.NewField(token.NoPos, nil, "Text", types.Typ[types.String], false))
	reply := markedType("Reply", local, types.NewField(token.NoPos, nil, "Text", text, false))
	badLocal := markedType("BadLocal", local, types.NewField(token.NoPos, nil, "Text", types.Typ[types.String], false))
	badSnapshot := markedType("BadSnapshot", snapshot, types.NewField(token.NoPos, nil, "Reply", reply, false))
	alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Alias", nil), reply)
	unchecked := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "UncheckedValue", nil),
		types.NewStruct([]*types.Var{types.NewField(token.NoPos, pkg, "value", types.NewPointer(retained), false)}, nil), nil)
	uncheckedAlias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Handle", nil), unchecked)
	namedHandles := types.NewNamed(types.NewTypeName(token.NoPos, nil, "Handles", nil), types.NewArray(unchecked, 2), nil)
	cases := []struct {
		typ                           types.Type
		snapshot, copied, read, local bool
	}{
		{unchecked, true, false, true, false},
		{uncheckedAlias, true, false, true, false},
		{types.NewArray(unchecked, 0), true, false, true, false},
		{types.NewArray(types.NewArray(uncheckedAlias, 1), 2), true, false, true, false},
		{namedHandles, false, false, false, false},
		{types.Typ[types.Int], true, true, true, false},
		{types.Typ[types.String], true, false, false, false},
		{frozenText, true, false, false, false},
		{copied, true, false, false, false},
		{snapshot, false, false, false, false},
		{local, false, false, false, true},
		{types.NewArray(copied, 0), true, false, false, false},
		{types.NewArray(types.Typ[types.Int], 2), true, true, true, false},
		{retained, true, false, false, false},
		{text, false, false, true, true},
		{reply, false, false, true, true},
		{alias, false, false, true, true},
		{badLocal, false, false, false, true},
		{badSnapshot, false, false, false, true},
		{types.NewArray(text, 0), false, false, true, true},
		{types.NewArray(types.Typ[types.String], 0), true, false, false, false},
		{types.NewPointer(reply), false, false, false, true},
		{types.NewSlice(text), false, false, false, true},
	}
	for _, reverse := range []bool{false, true} {
		var p Prover
		var q Classifier
		for index := range cases {
			if reverse {
				index = len(cases) - index - 1
			}
			test := cases[index]
			if got := p.FrozenReason(test.typ) == ""; got != test.snapshot {
				t.Errorf("Frozen(%s)=%v; want %v", test.typ, got, test.snapshot)
			}
			if got := p.ReadResultReason(test.typ) == ""; got != test.read {
				t.Errorf("ReadResult(%s)=%v; want %v", test.typ, got, test.read)
			}
			if got := p.CopiedReason(test.typ) == ""; got != test.copied {
				t.Errorf("Copied(%s)=%v; want %v", test.typ, got, test.copied)
			}
			if got := q.Locals(test.typ); got != test.local {
				t.Errorf("Locals(%s)=%v; want %v", test.typ, got, test.local)
			}
		}
	}
}
