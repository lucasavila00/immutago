package checker

import (
	"go/token"
	"go/types"
	"testing"
)

// TestDependencyClassification keeps the wrapper protected and its payload opaque.
func TestDependencyClassification(t *testing.T) {
	marker, _ := testTypes()
	value := markedType("Value", marker)
	payload := classificationSignature(value)
	pkg := types.NewPackage(RuntimePath, "proc")
	wrapper := types.NewNamed(types.NewTypeName(token.NoPos, pkg, UncheckedDependencyType, nil),
		types.NewStruct([]*types.Var{types.NewField(token.NoPos, pkg, "value", payload, false)}, nil), nil)
	alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Alias", nil), wrapper)
	var queries Classifier
	for _, typ := range []types.Type{wrapper, alias, types.NewArray(wrapper, 1), types.NewPointer(wrapper)} {
		if !queries.Contains(typ) || queries.Callable(typ) {
			t.Fatalf("wrong dependency classification for %s", typ)
		}
	}
	if !queries.Callable(classificationSignature(wrapper)) {
		t.Fatal("a callback that transports a dependency was not classified")
	}
	var proofs Prover
	if proofs.ArgumentReason(wrapper) != "" || proofs.Reason(wrapper) == "" {
		t.Fatal("a dependency must be an argument type but not an immutable stored type")
	}
}
