package checker

import (
	"go/token"
	"go/types"
	"testing"
)

// categoryCase records independently expected Frozen, Copied, and read-result membership.
type categoryCase struct {
	typ  types.Type
	want [3]bool
}

// TestCategoryValidity keeps category membership stable while failure paths change.
func TestCategoryValidity(t *testing.T) {
	cases := categoryCases(t)
	for _, bad := range []int{-1, 1} {
		for _, node := range stressGraph([][]int{{1}, {0}}, bad, true) {
			cases = append(cases, categoryCase{node, [3]bool{bad < 0, false, false}})
		}
	}
	for _, test := range append([]categoryCase(nil), cases...) {
		alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Alias", nil), test.typ)
		for _, typ := range []types.Type{alias, types.NewArray(alias, 0), types.NewArray(types.NewArray(alias, 2), 3)} {
			cases = append(cases, categoryCase{typ, test.want})
		}
		named := types.NewNamed(types.NewTypeName(token.NoPos, nil, "NamedArray", nil), types.NewArray(alias, 0), nil)
		cases = append(cases, categoryCase{named, [3]bool{}})
	}
	for _, reverse := range []bool{false, true} {
		var prover Prover
		for index := range cases {
			if reverse {
				index = len(cases) - index - 1
			}
			checkCategoryCase(t, &prover, cases[index])
		}
	}
	t.Logf("checked %d types in all three categories in both query orders", len(cases))
}

// checkCategoryCase checks all predicates twice, including cached failures and successful proofs.
func checkCategoryCase(t *testing.T, prover *Prover, test categoryCase) {
	t.Helper()
	for range 2 {
		for category, prove := range []func(types.Type) Failure{
			prover.FrozenFailure, prover.CopiedFailure, prover.ReadResultFailure,
		} {
			failure := prove(test.typ)
			if (failure.Reason == "") != test.want[category] {
				t.Fatalf("type %s, category %d: %v; want valid=%v", test.typ, category, failure, test.want[category])
			}
		}
	}
}

// categoryCases includes native, marked, opaque, recursive, and interface payloads.
func categoryCases(t *testing.T) []categoryCase {
	t.Helper()
	frozen, slice := testTypes()
	pkg := frozen.Obj().Pkg()
	local := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Local", nil), types.NewStruct(nil, nil), nil)
	copied := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Copied", nil), types.NewStruct(nil, nil), nil)
	text := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "StringProjection", nil), types.NewStruct(nil, nil), nil)
	frozenText := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "FrozenString", nil), types.NewStruct(nil, nil), nil)
	field := func(typ types.Type) *types.Var { return types.NewField(token.NoPos, nil, "Data", typ, false) }
	score := markedType("Score", copied, field(types.Typ[types.Int]))
	reply := markedType("Reply", local, field(text))
	settings := markedType("Settings", frozen, field(types.Typ[types.String]))
	badScore := markedType("BadScore", copied, field(types.NewArray(reply, 0)))
	unchecked := func(payload types.Type) types.Type {
		return types.NewNamed(types.NewTypeName(token.NoPos, pkg, "UncheckedValue", nil),
			types.NewStruct([]*types.Var{field(types.NewPointer(payload))}, nil), nil)
	}
	cases := []categoryCase{
		{types.Typ[types.Bool], [3]bool{true, true, true}}, {types.Typ[types.Int], [3]bool{true, true, true}},
		{types.Typ[types.String], [3]bool{true, false, false}}, {frozenText, [3]bool{true, false, false}},
		{score, [3]bool{true, true, true}}, {settings, [3]bool{true, false, false}},
		{text, [3]bool{false, false, true}}, {reply, [3]bool{false, false, true}},
		{types.NewPointer(settings), [3]bool{true, false, false}}, {types.NewPointer(reply), [3]bool{}},
		{unchecked(settings), [3]bool{true, false, true}}, {unchecked(reply), [3]bool{}},
		{unchecked(markedType("Opaque", frozen, field(types.NewSlice(reply)))), [3]bool{}},
		{badScore, [3]bool{}}, {markedType("OuterScore", copied, field(badScore)), [3]bool{}},
		{markedType("BadSettings", frozen, field(text)), [3]bool{}},
		{markedType("BadReply", local, field(types.Typ[types.String])), [3]bool{}},
		{types.NewSlice(reply), [3]bool{}}, {types.NewMap(types.Typ[types.String], reply), [3]bool{}},
		{sliceType(t, slice, reply), [3]bool{}}, {sliceType(t, slice, score), [3]bool{true, false, false}},
		{frozen, [3]bool{}}, {local, [3]bool{}}, {copied, [3]bool{}},
		{markedType("Mixed", copied, types.NewField(token.NoPos, nil, "Local", local, true)), [3]bool{}},
	}
	return append(cases, categoryReferenceCases(t, reply)...)
}

// categoryReferenceCases covers a Local method contract.
func categoryReferenceCases(t *testing.T, reply types.Type) []categoryCase {
	t.Helper()
	method := types.NewFunc(token.NoPos, nil, "Read", types.NewSignatureType(nil, nil, nil, nil,
		types.NewTuple(types.NewParam(token.NoPos, nil, "", reply)), false))
	return []categoryCase{
		{frozenContract("Reader", method), [3]bool{true, false, false}},
	}
}
