package checker

import (
	"go/token"
	"go/types"
	"strings"
	"testing"
)

// TestMarkerIdentities checks both exact library names, aliases, and same-named foreign types.
func TestMarkerIdentities(t *testing.T) {
	library := types.NewPackage(LibraryPath, "im")
	foreign := types.NewPackage("example.test/im", "im")
	for _, test := range []struct {
		name  string
		match func(types.Type) bool
	}{
		{"Shared", Shared},
		{"LocalRef", LocalRef},
	} {
		t.Run(test.name, func(t *testing.T) {
			marker := types.NewNamed(types.NewTypeName(token.NoPos, library, test.name, nil),
				types.NewStruct(nil, nil), nil)
			alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Alias", nil), marker)
			for _, field := range []types.Type{marker, alias} {
				if !test.match(markedType("Value", field)) {
					t.Fatal("marker identity lost through an alias")
				}
			}
			impostor := types.NewNamed(types.NewTypeName(token.NoPos, foreign, test.name, nil),
				marker.Underlying(), nil)
			defined := types.NewNamed(types.NewTypeName(token.NoPos, nil, "Defined", nil),
				marker.Underlying(), nil)
			for _, field := range []types.Type{impostor, defined, types.NewPointer(marker)} {
				if Marked(markedType("Value", field)) {
					t.Fatal("non-marker type established a marker policy")
				}
			}
		})
	}
}

// TestLocalPointerIdentity checks that aliases and defined pointer types retain the storage restriction.
func TestLocalPointerIdentity(t *testing.T) {
	library := types.NewPackage(LibraryPath, "im")
	marker := types.NewNamed(types.NewTypeName(token.NoPos, library, "LocalRef", nil),
		types.NewStruct(nil, nil), nil)
	value := markedType("Value", marker)
	pointer := types.NewPointer(value)
	alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Alias", nil), pointer)
	defined := types.NewNamed(types.NewTypeName(token.NoPos, nil, "Defined", nil), pointer, nil)
	for _, stored := range []types.Type{pointer, alias, defined} {
		if reason := Reason(stored); !strings.Contains(reason, "LocalRef pointers cannot be stored") {
			t.Errorf("local pointer %s accepted: %s", stored, reason)
		}
	}
	constraint := types.NewInterfaceType(nil, []types.Type{value}).Complete()
	parameter := types.NewTypeParam(types.NewTypeName(token.NoPos, nil, "T", nil), constraint)
	if reason := Reason(types.NewPointer(parameter)); !strings.Contains(reason, "concrete marked struct target") {
		t.Fatalf("type parameter admitted as a pointer target: %s", reason)
	}
}
