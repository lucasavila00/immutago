package checker

import (
	"go/token"
	"go/types"
	"testing"
)

// TestFrozenPointerCarriers checks recursive holders in both query orders.
func TestFrozenPointerCarriers(t *testing.T) {
	marker, _ := testTypes()
	record := markedType("Record", marker)
	pointer := types.NewPointer(record)
	alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Pointer", nil), pointer)
	constraint := types.NewInterfaceType(nil, []types.Type{
		types.NewUnion([]*types.Term{types.NewTerm(true, pointer)}),
	}).Complete()
	parameter := types.NewTypeParam(types.NewTypeName(token.NoPos, nil, "T", nil), constraint)
	first := types.NewNamed(types.NewTypeName(token.NoPos, nil, "First", nil), types.NewStruct(nil, nil), nil)
	second := types.NewNamed(types.NewTypeName(token.NoPos, nil, "Second", nil), types.NewStruct(nil, nil), nil)
	first.SetUnderlying(types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "Next", types.NewPointer(second), false),
		types.NewField(token.NoPos, nil, "Value", alias, false),
	}, nil))
	second.SetUnderlying(types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "Next", types.NewPointer(first), false),
	}, nil))
	shapes := []struct {
		typ      types.Type
		contains bool
	}{
		{record, true}, {pointer, false}, {alias, false}, {parameter, false},
		{first, false}, {second, false}, {types.NewPointer(first), false},
		{types.NewArray(pointer, 2), false}, {types.NewSlice(pointer), false},
		{types.NewMap(pointer, types.NewPointer(second)), false},
		{types.NewChan(types.SendRecv, pointer), false},
	}
	for _, reverse := range []bool{false, true} {
		var queries Classifier
		for step := range shapes {
			index := step
			if reverse {
				index = len(shapes) - step - 1
			}
			shape := shapes[index]
			if got := queries.Contains(shape.typ); got != shape.contains {
				t.Errorf("Contains(%s) = %t, want %t", shape.typ, got, shape.contains)
			}
		}
	}
}
