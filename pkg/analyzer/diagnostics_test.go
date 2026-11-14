package analyzer

import (
	"strings"
	"testing"
)

// TestStructuralDiagnosticContext checks the owner, failed field type, and supported repair.
func TestStructuralDiagnosticContext(t *testing.T) {
	cases := []struct {
		source string
		want   []string
	}{
		{`type Value struct{im.Shared; Items []string}`, []string{
			"immutable field Items", "owner: sample.Value; field type: []string",
			"fix: replace the native slice with im.FrozenSlice[T]",
		}},
		{`type Child struct{im.Shared}; type Parent struct{Child Child}`, []string{
			"unmarked struct", "owner: sample.Parent; field type: sample.Child",
			"fix: embed im.Shared",
		}},
		{`var _ = im.FreezeSlice([]*int{new(int)})`, []string{
			"immutable type argument", "type: *int", "fix: replace the pointer field or argument with an owned immutable value",
		}},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			got := checkSource(t, tc.source)
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Fatalf("missing %q in %s", want, got)
				}
			}
		})
	}
}

// TestDependencyWitnessSelection keeps the first cause without changing local errors or imported facts.
func TestDependencyWitnessSelection(t *testing.T) {
	failures := [...]proofFailure{
		{Location: "origin/a.go:1:1", Message: "rule A"},
		{Location: "origin/b.go:2:1", Message: "rule B"},
		{Location: "origin/a.go:1:1", Message: "rule A"},
	}
	before := failures
	original := invalidPackage{Relevant: true, Origin: "earlier", Failure: failures[0]}
	imports := map[string]invalidPackage{"wrapper": original}
	for range 10 {
		got := packageFailureFact("origin", false, imports, failures[:])
		if got.Relevant || got.Origin != "earlier" || got.Failure != failures[0] {
			t.Fatalf("first cause changed: %+v", got)
		}
		if failures != before || imports["wrapper"] != original {
			t.Fatal("selecting a witness changed an original error")
		}
	}
}

// TestFieldDiagnosticsUseOneDeclarationSite reports each failed field once across repeated type visits.
func TestFieldDiagnosticsUseOneDeclarationSite(t *testing.T) {
	got := checkSource(t, `type Value struct{im.Shared; Items []string; Pointer *int}; var _ Value`)
	for _, field := range []string{"Items", "Pointer"} {
		if count := strings.Count(got, "immutable field "+field+":"); count != 1 {
			t.Fatalf("field %s reported %d times: %s", field, count, got)
		}
	}
}

// TestPointerConstructorAcceptsCompilerBuiltinSignature checks the fresh allocation boundary.
func TestPointerConstructorAcceptsCompilerBuiltinSignature(t *testing.T) {
	got := checkSource(t, `type Value struct{im.Shared};func f(){_=new(Value)}`)
	if got != "" {
		t.Fatalf("incorrect constructor diagnostic: %s", got)
	}
}
