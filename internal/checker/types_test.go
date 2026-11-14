package checker

import (
	"go/token"
	"go/types"
	"strings"
	"testing"
)

// testTypes creates compiler type objects with the sealed library's exact package identity.
func testTypes() (*types.Named, *types.Named) {
	pkg := types.NewPackage(LibraryPath, "im")
	marker := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Shared", nil), types.NewStruct(nil, nil), nil)
	parameter := types.NewTypeParam(types.NewTypeName(token.NoPos, pkg, "T", nil),
		types.NewInterfaceType(nil, nil).Complete())
	slice := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "FrozenSlice", nil),
		types.NewStruct([]*types.Var{types.NewField(token.NoPos, pkg, "values", types.NewSlice(parameter), false)}, nil), nil)
	slice.SetTypeParams([]*types.TypeParam{parameter})
	return marker, slice
}

// markedType creates a named marked struct whose fields can later be set for a recursive test.
func markedType(name string, marker types.Type, fields ...*types.Var) *types.Named {
	fields = append([]*types.Var{types.NewField(token.NoPos, nil, "Shared", marker, true)}, fields...)
	return types.NewNamed(types.NewTypeName(token.NoPos, nil, name, nil), types.NewStruct(fields, nil), nil)
}

// sliceType instantiates a sealed slice with one element obligation.
func sliceType(t *testing.T, slice *types.Named, element types.Type) types.Type {
	t.Helper()
	result, err := types.Instantiate(nil, slice, []types.Type{element}, true)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

// TestProofGraph checks recursive safe graphs and failures reachable after a recursive edge.
func TestProofGraph(t *testing.T) {
	marker, slice := testTypes()
	good := markedType("Good", marker)
	good.SetUnderlying(types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "Shared", marker, true),
		types.NewField(token.NoPos, nil, "Next", sliceType(t, slice, good), false),
	}, nil))
	if reason := Reason(good); reason != "" {
		t.Fatalf("safe recursion rejected: %s", reason)
	}
	bad := markedType("Bad", marker)
	bad.SetUnderlying(types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "Shared", marker, true),
		types.NewField(token.NoPos, nil, "Next", sliceType(t, slice, bad), false),
		types.NewField(token.NoPos, nil, "Mutable", types.NewPointer(types.Typ[types.Int]), false),
	}, nil))
	if reason := Reason(bad); !strings.Contains(reason, "Mutable") {
		t.Fatalf("bad recursion accepted: %q", reason)
	}
	if reason := Reason(sliceType(t, slice, bad)); !strings.Contains(reason, "Mutable") {
		t.Fatalf("bad collection recursion accepted: %q", reason)
	}
}

// TestPrimitiveProofs checks each supported primitive and rejects mutable runtime storage.
func TestPrimitiveProofs(t *testing.T) {
	for _, kind := range []types.BasicKind{
		types.Bool, types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
		types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr,
		types.Float32, types.Float64, types.Complex64, types.Complex128, types.String,
	} {
		if reason := Reason(types.Typ[kind]); reason != "" {
			t.Errorf("%s: %s", types.Typ[kind], reason)
		}
	}
	for _, typ := range []types.Type{
		types.Typ[types.UnsafePointer], types.NewSlice(types.Typ[types.Int]),
		types.NewMap(types.Typ[types.String], types.Typ[types.Int]),
		types.NewChan(types.SendRecv, types.Typ[types.Int]),
		types.NewPointer(types.Typ[types.Int]), types.NewStruct(nil, nil),
	} {
		if reason := Reason(typ); reason == "" {
			t.Errorf("accepted mutable or unmarked type %s", typ)
		}
	}
	if reason := Reason(types.NewArray(types.Typ[types.Int], 3)); reason != "" {
		t.Fatal(reason)
	}
}

// TestOwnershipGraph checks containers, recursive unmarked structs, aliases, and exact marker identity.
func TestOwnershipGraph(t *testing.T) {
	marker, slice := testTypes()
	value := markedType("Value", marker)
	for _, typ := range []types.Type{
		value, types.NewPointer(value), types.NewSlice(value), types.NewArray(value, 2),
		types.NewMap(value, types.Typ[types.Int]), types.NewChan(types.SendRecv, value),
		sliceType(t, slice, types.Typ[types.Int]),
	} {
		if !Contains(typ) {
			t.Errorf("missed immutable storage in %s", typ)
		}
	}
	node := types.NewNamed(types.NewTypeName(token.NoPos, nil, "Node", nil), types.NewStruct(nil, nil), nil)
	node.SetUnderlying(types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "Next", types.NewPointer(node), false),
	}, nil))
	if Contains(node) {
		t.Fatal("ordinary recursive type contains immutable storage")
	}
	other := types.NewNamed(types.NewTypeName(token.NoPos, types.NewPackage("other", "im"), "Shared", nil),
		types.NewStruct(nil, nil), nil)
	if Marked(markedType("Pretender", other)) {
		t.Fatal("accepted marker from another package")
	}
	alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Alias", nil), marker)
	if !Marked(markedType("Aliased", alias)) {
		t.Fatal("failed to resolve marker alias")
	}
	if Contains(nil) || Marked(nil) {
		t.Fatal("nil type contains immutable storage")
	}
}

// TestSharedProofGraph verifies that a deep shared DAG creates one obligation per type, not per path.
func TestSharedProofGraph(t *testing.T) {
	marker, _ := testTypes()
	var current types.Type = types.Typ[types.Int]
	for depth := 0; depth < 45; depth++ {
		current = markedType("Layer", marker,
			types.NewField(token.NoPos, nil, "Left", current, false),
			types.NewField(token.NoPos, nil, "Right", current, false))
	}
	p := &proof{nodes: make(map[types.Type]*obligation)}
	p.node(current)
	if len(p.nodes) != 47 {
		t.Fatalf("built %d obligations; want 47", len(p.nodes))
	}
	if reason := Reason(current); reason != "" {
		t.Fatal(reason)
	}
}

// TestNamedArraysRequireMarker keeps custom aggregates inside the marked-struct contract.
func TestNamedArraysRequireMarker(t *testing.T) {
	array := types.NewArray(types.Typ[types.Int], 2)
	named := types.NewNamed(types.NewTypeName(token.NoPos, nil, "Count", nil), array, nil)
	if reason := Reason(named); !strings.Contains(reason, "named array") {
		t.Fatalf("named array proof: %q", reason)
	}
	alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Pair", nil), array)
	if reason := Reason(alias); reason != "" {
		t.Fatalf("array alias: %s", reason)
	}
	marker, _ := testTypes()
	wrapped := markedType("Count", marker, types.NewField(token.NoPos, nil, "Values", array, false))
	if reason := Reason(wrapped); reason != "" {
		t.Fatalf("marked wrapper: %s", reason)
	}
	union := types.NewUnion([]*types.Term{types.NewTerm(true, array)})
	constraint := types.NewInterfaceType(nil, []types.Type{union}).Complete()
	parameter := types.NewTypeParam(types.NewTypeName(token.NoPos, nil, "T", nil), constraint)
	if reason := Reason(parameter); reason == "" {
		t.Fatalf("tilde array proof: %q", reason)
	}
}
