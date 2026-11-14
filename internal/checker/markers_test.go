package checker

import (
	"go/token"
	"go/types"
	"strings"
	"testing"
)

// TestMarkerReferenceProofs checks the two marker modes and stored reference boundaries.
func TestMarkerReferenceProofs(t *testing.T) {
	sharedMarker, slice := testTypes()
	localMarker := types.NewNamed(types.NewTypeName(token.NoPos, sharedMarker.Obj().Pkg(), "Scoped", nil),
		types.NewStruct(nil, nil), nil)
	shared := markedType("Shared", sharedMarker)
	local := markedType("Local", localMarker)
	if !Shared(shared) || Scoped(shared) || !Scoped(local) || Shared(local) {
		t.Fatal("marker modes were not preserved")
	}
	for _, value := range []types.Type{
		shared, local, types.NewPointer(shared), types.NewArray(local, 2), sliceType(t, slice, local),
		markedType("Parent", sharedMarker, types.NewField(token.NoPos, nil, "Child", local, false)),
		markedType("Reader", localMarker, types.NewField(token.NoPos, nil, "Child", types.NewPointer(shared), false)),
	} {
		if reason := Reason(value); reason != "" {
			t.Errorf("immutable value %s rejected: %s", value, reason)
		}
	}
	for _, value := range []types.Type{
		types.NewPointer(local), types.NewArray(types.NewPointer(local), 2),
		sliceType(t, slice, types.NewPointer(local)),
		markedType("Bad", sharedMarker, types.NewField(token.NoPos, nil, "Child", types.NewPointer(local), false)),
	} {
		if reason := Reason(value); !strings.Contains(reason, "Scoped pointers cannot be stored") {
			t.Errorf("stored local reference %s: %s", value, reason)
		}
	}
	mixed := markedType("Mixed", sharedMarker,
		types.NewField(token.NoPos, nil, "Scoped", localMarker, true))
	if reason := Reason(mixed); !strings.Contains(reason, "cannot embed both") {
		t.Fatalf("mixed markers accepted: %s", reason)
	}
}

// TestRecursivePointerProof checks cycles through stored shared references without skipping a bad field.
func TestRecursivePointerProof(t *testing.T) {
	marker, _ := testTypes()
	node := markedType("Node", marker)
	fields := []*types.Var{
		types.NewField(token.NoPos, nil, "Shared", marker, true),
		types.NewField(token.NoPos, nil, "Next", types.NewPointer(node), false),
	}
	node.SetUnderlying(types.NewStruct(fields, nil))
	if reason := Reason(node); reason != "" {
		t.Fatal(reason)
	}
	fields = append(fields, types.NewField(token.NoPos, nil, "Mutable", types.NewPointer(types.Typ[types.Int]), false))
	node.SetUnderlying(types.NewStruct(fields, nil))
	if reason := Reason(node); !strings.Contains(reason, "Mutable") {
		t.Fatalf("mutable recursive field accepted: %s", reason)
	}
}

// TestServiceProofs checks that a Ref is an allowed stored value and a service is not.
func TestServiceProofs(t *testing.T) {
	sharedMarker, _ := testTypes()
	runtime := types.NewPackage(RuntimePath, "supervise")
	serviceMarker := types.NewNamed(types.NewTypeName(token.NoPos, runtime, "Service", nil),
		types.NewStruct(nil, nil), nil)
	service := types.NewNamed(types.NewTypeName(token.NoPos, nil, "ConfigLoader", nil), types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "Service", serviceMarker, true),
		types.NewField(token.NoPos, nil, "version", types.Typ[types.Int], false),
	}, nil), nil)
	ref := types.NewNamed(types.NewTypeName(token.NoPos, runtime, "Ref", nil), types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, runtime, "value", types.NewPointer(service), false),
	}, nil), nil)
	alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "LoaderRef", nil), ref)
	if !Service(service) || Marked(service) || MarkerCount(service) != 0 ||
		!IsRuntimeType(alias, RefType) || IsRuntimeType(service, RefType) {
		t.Fatal("service identities were not preserved")
	}
	for _, value := range []types.Type{
		ref, alias, types.NewArray(ref, 2),
		markedType("Peers", sharedMarker, types.NewField(token.NoPos, nil, "Config", ref, false)),
	} {
		if reason := Reason(value); reason != "" {
			t.Errorf("Ref value %s rejected: %s", value, reason)
		}
	}
	for _, value := range []types.Type{service, types.NewPointer(service)} {
		if Reason(value) == "" {
			t.Errorf("service value %s accepted", value)
		}
	}
}
