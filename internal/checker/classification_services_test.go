package checker

import (
	"go/token"
	"go/types"
	"slices"
	"testing"
)

// TestServicePointerClassification checks package identity, aliases, containers, and query order.
func TestServicePointerClassification(t *testing.T) {
	runtime := types.NewPackage(RuntimePath, "supervise")
	marker := types.NewNamed(types.NewTypeName(token.NoPos, runtime, "Service", nil), types.NewStruct(nil, nil), nil)
	owner := types.NewPackage("example.com/service", "service")
	consumer := types.NewPackage("example.com/consumer", "consumer")
	structure := types.NewStruct([]*types.Var{types.NewField(token.NoPos, owner, "Service", marker, true)}, nil)
	service := types.NewNamed(types.NewTypeName(token.NoPos, owner, "Worker", nil), structure, nil)
	alias := types.NewAlias(types.NewTypeName(token.NoPos, consumer, "Alias", nil), service)
	pointer := types.NewPointer(service)
	for _, current := range []*types.Package{nil, owner, consumer} {
		for _, reverse := range []bool{false, true} {
			queries := Classifier{Package: current}
			typesToCheck := []types.Type{pointer, types.NewPointer(alias), types.NewArray(pointer, 1)}
			if reverse {
				slices.Reverse(typesToCheck)
			}
			want := current == consumer
			for _, typ := range typesToCheck {
				if queries.Contains(typ) != want || queries.Owns(typ) != want ||
					queries.Transport(typ) != want || queries.Locals(typ) != want {
					t.Fatalf("package=%v type=%s: inconsistent Local classification", current, typ)
				}
			}
			if queries.ServicePointer(types.NewPointer(structure)) != (current != nil) {
				t.Fatalf("package=%v: anonymous service pointer has a declaring package", current)
			}
		}
	}
}

// TestServicePointerProducerContracts keeps pointer producers ordinary and their results Local.
func TestServicePointerProducerContracts(t *testing.T) {
	runtime := types.NewPackage(RuntimePath, "supervise")
	marker := types.NewNamed(types.NewTypeName(token.NoPos, runtime, "Service", nil), types.NewStruct(nil, nil), nil)
	owner := types.NewPackage("example.com/service", "service")
	consumer := types.NewPackage("example.com/consumer", "consumer")
	structure := types.NewStruct([]*types.Var{types.NewField(token.NoPos, owner, "Service", marker, true)}, nil)
	service := types.NewNamed(types.NewTypeName(token.NoPos, owner, "Worker", nil), structure, nil)
	pointer := types.NewPointer(service)
	local := types.NewNamed(types.NewTypeName(token.NoPos, types.NewPackage(LibraryPath, "im"),
		"StringProjection", nil), types.NewStruct(nil, nil), nil)
	for _, test := range []struct {
		name       string
		parameters []types.Type
		results    []types.Type
		restricted bool
	}{
		{"producer", nil, []types.Type{pointer}, false},
		{"parameter", []types.Type{pointer}, nil, true},
		{"mixed results", nil, []types.Type{pointer, local}, true},
		{"Local result", nil, []types.Type{local}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			queries := Classifier{Package: consumer}
			signature := types.NewSignatureType(nil, nil, nil,
				serviceContractTuple(test.parameters), serviceContractTuple(test.results), false)
			if queries.Callable(signature) != test.restricted || !queries.LocalPointer(pointer) {
				t.Fatal("producer contract changed the Local pointer or callback rule")
			}
		})
	}
}

// serviceContractTuple makes a compiler contract without source-name dependencies.
func serviceContractTuple(items []types.Type) *types.Tuple {
	variables := make([]*types.Var, len(items))
	for index, typ := range items {
		variables[index] = types.NewVar(token.NoPos, nil, "", typ)
	}
	return types.NewTuple(variables...)
}
