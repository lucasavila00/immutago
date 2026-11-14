package analyzer

import (
	"fmt"
	"testing"
)

// TestNilBuilderForms applies the nil source rule across types, syntax, and binding sites.
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
		{"marked values", "type V struct{ im.Shared }", "[]V", "FreezeSlice"},
		{"shared pointers", "type V struct{ im.Shared }", "[]*V", "FreezeSlice"},
		{"local values", "type V struct{ im.LocalRef }", "[]V", "FreezeSlice"},
		{"marked map", "type V struct{ im.Shared }", "map[*V]V", "FreezeMap"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, syntax := range []string{"%s(nil)", "(%s)((nil))"} {
				checkNilBuilderSites(t, tc.declarations, tc.target, fmt.Sprintf(syntax, tc.target), tc.freeze)
			}
		})
	}
}

// checkNilBuilderSites checks the same source at each documented construction boundary.
func checkNilBuilderSites(t *testing.T, declarations, target, source, freeze string) {
	t.Helper()
	for _, body := range []string{
		"values := %s; _ = im.%s(values)",
		"var values = %s; _ = im.%s(values)",
		"var values " + target + " = %s; _ = im.%s(values)",
		"_ = im.%[2]s(%[1]s)",
	} {
		program := declarations + "\nfunc f(){" + fmt.Sprintf(body, source, freeze) + "}"
		if got := checkSource(t, program); got != "" {
			t.Fatalf("allowed nil construction rejected:\n%s\n%s", program, got)
		}
	}
}
