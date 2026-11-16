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
	localMarker := types.NewNamed(types.NewTypeName(token.NoPos, sharedMarker.Obj().Pkg(), "Local", nil),
		types.NewStruct(nil, nil), nil)
	shared := markedType("Frozen", sharedMarker)
	local := markedType("Local", localMarker)
	if !Frozen(shared) || Local(shared) || !Local(local) || Frozen(local) {
		t.Fatal("marker modes were not preserved")
	}
	for _, value := range []types.Type{
		shared, local, types.NewPointer(shared), types.NewArray(local, 2), sliceType(t, slice, shared),
		markedType("Parent", sharedMarker, types.NewField(token.NoPos, nil, "Child", shared, false)),
		markedType("Reader", localMarker, types.NewField(token.NoPos, nil, "Child", local, false)),
	} {
		if reason := Reason(value); reason != "" {
			t.Errorf("immutable value %s rejected: %s", value, reason)
		}
	}
	for _, value := range []types.Type{
		types.NewPointer(local), types.NewArray(types.NewPointer(local), 2),
		sliceType(t, slice, types.NewPointer(local)),
		sliceType(t, slice, local),
		markedType("Bad", sharedMarker, types.NewField(token.NoPos, nil, "Child", types.NewPointer(local), false)),
		markedType("BadValue", sharedMarker, types.NewField(token.NoPos, nil, "Child", local, false)),
		markedType("BadReader", localMarker, types.NewField(token.NoPos, nil, "Child", types.NewPointer(shared), false)),
		markedType("BadReaderValue", localMarker, types.NewField(token.NoPos, nil, "Child", shared, false)),
	} {
		if reason := Reason(value); reason == "" {
			t.Errorf("stored local reference %s: %s", value, reason)
		}
	}
	mixed := markedType("Mixed", sharedMarker,
		types.NewField(token.NoPos, nil, "Local", localMarker, true))
	if reason := Reason(mixed); reason != "struct cannot embed both im.Frozen and im.Local" {
		t.Fatalf("mixed markers accepted: %s", reason)
	}
}

// TestRecursivePointerProof checks cycles through stored shared references without skipping a bad field.
func TestRecursivePointerProof(t *testing.T) {
	marker, _ := testTypes()
	node := markedType("Node", marker)
	fields := []*types.Var{
		types.NewField(token.NoPos, nil, "Frozen", marker, true),
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
