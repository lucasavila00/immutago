package analyzer

import (
	"go/types"
	"testing"
)

// TestTestMainPackage checks the compiler identity required for a generated test runner.
func TestTestMainPackage(t *testing.T) {
	cases := []struct {
		path, name, imported string
		want                 bool
	}{
		{"example/p.test", "main", "testing/internal/testdeps", true},
		{"example/p.test", "main", "testing", false},
		{"example/p.test", "p", "testing/internal/testdeps", false},
		{"example/p", "main", "testing/internal/testdeps", false},
	}
	for _, tc := range cases {
		pkg := types.NewPackage(tc.path, tc.name)
		pkg.SetImports([]*types.Package{types.NewPackage(tc.imported, "dependency")})
		if got := testMainPackage(pkg); got != tc.want {
			t.Errorf("%s (%s), import %s: got %t, want %t", tc.path, tc.name, tc.imported, got, tc.want)
		}
	}
}
