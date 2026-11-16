package checker

import (
	"go/token"
	"go/types"
	"strings"
	"testing"
)

// TestMarkerIdentities checks exact library names, aliases, and same-named foreign types.
func TestMarkerIdentities(t *testing.T) {
	library := types.NewPackage(LibraryPath, "im")
	foreign := types.NewPackage("example.test/im", "im")
	for _, test := range []struct {
		name  string
		match func(types.Type) bool
	}{
		{"Frozen", Frozen},
		{"Local", Local},
		{"Copied", Copied},
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
				if value := markedType("Value", field); Marked(value) || Copied(value) {
					t.Fatal("non-marker type established a marker policy")
				}
			}
			duplicate := markedType("Duplicate", marker, types.NewField(token.NoPos, nil, "Alias", alias, true))
			if MarkerCount(duplicate) != 2 || !strings.Contains(Reason(duplicate), "exactly one") {
				t.Fatal("duplicate aliases passed the structural proof")
			}
		})
	}
}

// TestMarkerFailures uses one reason for every category, including alias and mixed markers.
func TestMarkerFailures(t *testing.T) {
	library := types.NewPackage(LibraryPath, "im")
	markers := make(map[string]types.Type)
	for _, name := range []string{"Frozen", "Local", "Copied"} {
		markers[name] = types.NewNamed(types.NewTypeName(token.NoPos, library, name, nil),
			types.NewStruct(nil, nil), nil)
	}
	for _, pair := range [][2]string{
		{"Frozen", "Frozen"}, {"Local", "Local"}, {"Copied", "Copied"},
		{"Frozen", "Local"}, {"Local", "Frozen"}, {"Frozen", "Copied"},
		{"Copied", "Frozen"}, {"Local", "Copied"}, {"Copied", "Local"},
	} {
		t.Run(pair[0]+"_"+pair[1], func(t *testing.T) {
			alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Alias", nil), markers[pair[1]])
			value := markedType("Mixed", markers[pair[0]], types.NewField(token.NoPos, nil, "Alias", alias, true))
			want := "struct must embed exactly one immutable marker; each embedded alias counts as a marker"
			if pair == [2]string{"Frozen", "Local"} || pair == [2]string{"Local", "Frozen"} {
				want = "struct cannot embed both im.Frozen and im.Local"
			}
			var prover Prover
			checks := []func(types.Type) string{
				prover.FrozenReason, prover.ReadResultReason, prover.CopiedReason, prover.Reason,
			}
			for range 2 {
				for _, check := range checks {
					if got := check(value); got != want {
						t.Errorf("marker reason = %q; want %q", got, want)
					}
				}
			}
		})
	}
}

// TestLocalPointerIdentity checks that aliases and defined pointer types retain the storage restriction.
func TestLocalPointerIdentity(t *testing.T) {
	library := types.NewPackage(LibraryPath, "im")
	marker := types.NewNamed(types.NewTypeName(token.NoPos, library, "Local", nil),
		types.NewStruct(nil, nil), nil)
	value := markedType("Value", marker)
	pointer := types.NewPointer(value)
	alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Alias", nil), pointer)
	defined := types.NewNamed(types.NewTypeName(token.NoPos, nil, "Defined", nil), pointer, nil)
	for _, stored := range []types.Type{pointer, alias, defined} {
		if reason := Reason(stored); reason == "" {
			t.Errorf("local pointer %s accepted: %s", stored, reason)
		}
	}
	constraint := types.NewInterfaceType(nil, []types.Type{value}).Complete()
	parameter := types.NewTypeParam(types.NewTypeName(token.NoPos, nil, "T", nil), constraint)
	if reason := Reason(types.NewPointer(parameter)); !strings.Contains(reason, "concrete Frozen record target") {
		t.Fatalf("type parameter admitted as a pointer target: %s", reason)
	}
}
