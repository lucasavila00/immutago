package analyzer

import (
	"fmt"
	"strings"
	"testing"
)

// TestNilBuilderForms accepts bare nil declarations and rejects conversion sources at each binding site.
func TestNilBuilderForms(t *testing.T) {
	cases := []struct {
		name, declarations, target, freeze string
	}{
		{"slice", "", "[]int", "FreezeSlice"},
		{"map", "", "map[string]int", "FreezeMap"},
		{"named slice", "type Ints []int", "Ints", "FreezeSlice"},
		{"named map", "type Scores map[string]int", "Scores", "FreezeMap"},
		{"alias slice", "type Ints = []int", "Ints", "FreezeSlice"},
		{"alias map", "type Scores = map[string]int", "Scores", "FreezeMap"},
		{"generic scalar slice", "type List[T any] []T", "List[int]", "FreezeSlice"},
		{"marked values", "type V struct{ im.Frozen }", "[]V", "FreezeSlice"},
		{"shared pointers", "type V struct{ im.Frozen }", "[]*V", "FreezeSlice"},
		{"marked map", "type V struct{ im.Frozen }", "map[*V]V", "FreezeMap"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, initializer := range []string{"", " = nil"} {
				program := tc.declarations + "\nfunc f(){var values " + tc.target + initializer +
					"; _ = im." + tc.freeze + "(values)}"
				if got := checkSource(t, program); got != "" {
					t.Fatalf("allowed nil declaration rejected:\n%s\n%s", program, got)
				}
			}
			for _, syntax := range []string{"%s(nil)", "(%s)((nil))"} {
				rejectNilConversionSites(t, tc.declarations, tc.target, fmt.Sprintf(syntax, tc.target), tc.freeze)
			}
		})
	}
}

// rejectNilConversionSites rejects conversion syntax at every checked construction boundary.
func rejectNilConversionSites(t *testing.T, declarations, target, source, freeze string) {
	t.Helper()
	for _, test := range []struct{ body, want string }{
		{"values := %s; _ = im.%s(values)", "native builder values requires a fresh local"},
		{"var values = %s; _ = im.%s(values)", "native builder values requires a fresh local"},
		{"var values " + target + " = %s; _ = im.%s(values)", "native builder values requires a fresh local"},
		{"_ = im.%[2]s(%[1]s)", "cannot freeze"},
	} {
		program := declarations + "\nfunc f(){" + fmt.Sprintf(test.body, source, freeze) + "}"
		if got := checkSource(t, program); !strings.Contains(got, test.want) {
			t.Fatalf("missing conversion-source rejection:\n%s\n%s", program, got)
		}
	}
}
