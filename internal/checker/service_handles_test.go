package checker

import (
	"go/token"
	"go/types"
	"testing"
)

// TestClassifierHandles checks handle storage and callable contracts in both query orders.
func TestClassifierHandles(t *testing.T) {
	marker, _ := testTypes()
	value := markedType("Value", marker)
	runtime := types.NewPackage(RuntimePath, "supervise")
	fields := []*types.Var{
		types.NewField(token.NoPos, runtime, "value", types.NewPointer(value), false),
		types.NewField(token.NoPos, runtime, "callback", classificationSignature(value), false),
	}
	ref := types.NewNamed(types.NewTypeName(token.NoPos, runtime, "Ref", nil), types.NewStruct(fields, nil), nil)
	alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Alias", nil), ref)
	results := types.NewTuple(types.NewVar(token.NoPos, nil, "", alias))
	producer := types.NewSignatureType(nil, nil, nil, nil, results, false)
	contract := types.NewInterfaceType([]*types.Func{types.NewFunc(token.NoPos, nil, "Worker", producer)}, nil)
	nodes := make([]*types.Named, 3)
	for i, name := range []string{"First", "Second", "Empty"} {
		nodes[i] = types.NewNamed(types.NewTypeName(token.NoPos, nil, name, nil), types.NewStruct(nil, nil), nil)
	}
	nodes[0].SetUnderlying(types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "Next", types.NewPointer(nodes[1]), false),
	}, nil))
	nodes[1].SetUnderlying(types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "Next", types.NewPointer(nodes[0]), false),
		types.NewField(token.NoPos, nil, "Handle", alias, false),
	}, nil))
	nodes[2].SetUnderlying(types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "Next", types.NewPointer(nodes[2]), false),
	}, nil))
	shapes := []types.Type{
		ref, alias, types.NewPointer(alias), types.NewArray(alias, 0), types.NewSlice(alias),
		types.NewMap(alias, types.Typ[types.Int]), types.NewChan(types.SendRecv, alias),
		classificationSignature(alias), producer, contract.Complete(), nodes[0], nodes[1],
	}
	for _, reverse := range []bool{false, true} {
		var queries Classifier
		for index := range shapes {
			if reverse {
				index = len(shapes) - 1 - index
			}
			typ := shapes[index]
			if queries.Handles(nodes[2]) || queries.Handles(classificationSignature(types.Typ[types.Int])) {
				t.Fatal("ordinary recursive storage or callback contains a handle")
			}
			if !queries.Handles(typ) {
				t.Errorf("handle not found in %s", typ)
			}
			if queries.Contains(typ) || queries.Callable(typ) {
				t.Errorf("Ref implementation is not opaque in %s", typ)
			}
		}
	}
}
