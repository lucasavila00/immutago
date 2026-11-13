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
		{`type Value struct{im.Marker; Items []string}`, []string{
			"immutable field Items", "owner: sample.Value; field type: []string",
			"fix: replace the native slice with im.Slice[T]",
		}},
		{`type Child struct{im.Marker}; type Parent struct{Child Child}`, []string{
			"unmarked struct", "owner: sample.Parent; field type: sample.Child",
			"fix: embed im.Marker",
		}},
		{`var _ = im.NewSlice(new(int))`, []string{
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

// TestDependencyFailureDeduplication keeps every original cause without expanding diamond import routes.
func TestDependencyFailureDeduplication(t *testing.T) {
	failures := []proofFailure{
		{Location: "origin/a.go:1:1", Message: "rule A", Chain: []string{"consumer", "z", "origin"}},
		{Location: "origin/b.go:2:1", Message: "rule B", Chain: []string{"consumer", "origin"}},
		{Location: "origin/a.go:1:1", Message: "rule A", Chain: []string{"consumer", "a", "origin"}},
	}
	got := uniqueFailures(failures)
	if len(got) != 2 || got[0].Chain[1] != "a" || got[1].Message != "rule B" {
		t.Fatalf("failure causes or deterministic routes changed: %+v", got)
	}
}

// TestFieldDiagnosticsUseOneDeclarationSite reports each failed field once across repeated type visits.
func TestFieldDiagnosticsUseOneDeclarationSite(t *testing.T) {
	got := checkSource(t, `type Value struct{im.Marker; Items []string; Pointer *int}; var _ Value`)
	for _, field := range []string{"Items", "Pointer"} {
		if count := strings.Count(got, "immutable field "+field+":"); count != 1 {
			t.Fatalf("field %s reported %d times: %s", field, count, got)
		}
	}
}

// TestPointerConstructorDiagnosticDoesNotClassifyItsTypeAsAnArgument keeps the actionable allocation error.
func TestPointerConstructorDiagnosticDoesNotClassifyItsTypeAsAnArgument(t *testing.T) {
	got := checkSource(t, `type Value struct{im.Marker};func f(){_=new(Value)}`)
	if !strings.Contains(got, "pointer-owned construction") || strings.Contains(got, "dynamic calls") {
		t.Fatalf("incorrect constructor diagnostic: %s", got)
	}
}
