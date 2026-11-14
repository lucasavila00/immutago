package analyzer

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/lucasavila00/immutago/internal/checker"
	"golang.org/x/tools/go/analysis"
)

// dependencyImporter supplies the preceding packages in the failed-proof import chain.
type dependencyImporter map[string]*types.Package

// Import returns compiler identities shared by the package fact test.
func (packages dependencyImporter) Import(path string) (*types.Package, error) {
	return packages[path], nil
}

// TestInvalidDependencyProofPropagates checks local diagnostics, silent wrappers, and relevant consumers.
func TestInvalidDependencyProofPropagates(t *testing.T) {
	for _, serialized := range []bool{false, true} {
		t.Run(fmt.Sprintf("serialized=%t", serialized), func(t *testing.T) {
			checkInvalidDependencyPropagation(t, serialized)
		})
	}
}

// checkInvalidDependencyPropagation supplies only direct facts, as the vet driver does.
func checkInvalidDependencyPropagation(t *testing.T, serialized bool) {
	t.Helper()
	library := types.NewPackage(checker.LibraryPath, "im")
	library.MarkComplete()
	packages := dependencyImporter{"unsafe": types.Unsafe, checker.LibraryPath: library}
	invalid := make(map[*types.Package]invalidPackage)
	cases := []struct{ path, source, diagnostic string }{
		{"example/bad", `package bad; import "unsafe"; var _ unsafe.Pointer`, "unsupported unsafe"},
		{"example/wrapper", `package wrapper; import _ "example/bad"`, ""},
		{"example/consumer", `package consumer; import (_ "example/wrapper"; _ "` + checker.LibraryPath + `")`,
			"unsupported dependency example/wrapper"},
	}
	for _, tc := range cases {
		positions := token.NewFileSet()
		file, err := parser.ParseFile(positions, "proof.go", tc.source, 0)
		if err != nil {
			t.Fatal(err)
		}
		info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue), Defs: make(map[*ast.Ident]types.Object),
			Uses: make(map[*ast.Ident]types.Object), Scopes: make(map[ast.Node]*types.Scope),
			Implicits: make(map[ast.Node]types.Object)}
		config := types.Config{Importer: packages}
		pkg, err := config.Check(tc.path, positions, []*ast.File{file}, info)
		if err != nil {
			t.Fatal(err)
		}
		available := make(map[*types.Package]invalidPackage)
		for _, imported := range pkg.Imports() {
			if fact, ok := invalid[imported]; ok {
				if serialized {
					fact = *roundTripFact(t, &fact)
				}
				available[imported] = fact
			}
		}
		var reports []string
		pass := &analysis.Pass{Analyzer: Analyzer, Fset: positions, Pkg: pkg, Files: []*ast.File{file}, TypesInfo: info,
			Report: func(d analysis.Diagnostic) { reports = append(reports, d.Message) },
			ImportPackageFact: func(p *types.Package, fact analysis.Fact) bool {
				value, ok := available[p]
				*fact.(*invalidPackage) = value
				return ok
			},
			ExportPackageFact: func(fact analysis.Fact) { invalid[pkg] = *fact.(*invalidPackage) }}
		if _, err := run(pass); err != nil {
			t.Fatal(err)
		}
		if _, ok := invalid[pkg]; !ok {
			t.Fatalf("%s did not export its failed proof", tc.path)
		}
		got := strings.Join(reports, "\n")
		if tc.diagnostic == "" && got != "" || tc.diagnostic != "" && !strings.Contains(got, tc.diagnostic) {
			t.Errorf("%s diagnostics = %q, want %q", tc.path, got, tc.diagnostic)
		}
		if tc.path == "example/consumer" {
			checkDependencyContext(t, got)
		}

		packages[tc.path] = pkg
	}
}

// TestInvalidProofKeepsRelevanceWhenVetPrunesImports checks the scope bit carried by package facts.
func TestInvalidProofKeepsRelevanceWhenVetPrunesImports(t *testing.T) {
	positions := token.NewFileSet()
	file, err := parser.ParseFile(positions, "wrapper.go", `package wrapper; import _ "example/bad"`, 0)
	if err != nil {
		t.Fatal(err)
	}
	bad := types.NewPackage("example/bad", "bad")
	wrapper := types.NewPackage("example/wrapper", "wrapper")
	wrapper.SetImports([]*types.Package{bad})
	var reports []string
	pass := &analysis.Pass{Pkg: wrapper, Fset: positions, Files: []*ast.File{file}, TypesInfo: &types.Info{},
		ImportPackageFact: func(_ *types.Package, fact analysis.Fact) bool {
			*fact.(*invalidPackage) = *packageFailureFact("example/bad", true, nil, nil)
			return true
		},
		Report: func(d analysis.Diagnostic) { reports = append(reports, d.Message) }}
	invalid, relevant, _ := invalidDependencies(pass)
	if !invalid || !relevant || len(reports) != 1 {
		t.Fatal("vet's pruned import graph lost a dependency proof failure")
	}
}

// checkDependencyContext checks the portable source and import route of the original failure.
func checkDependencyContext(t *testing.T, got string) {
	t.Helper()
	for _, want := range []string{
		"example/consumer -> example/wrapper -> example/bad", "example/bad/proof.go:1:",
		"unsupported unsafe import", "fix:",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing dependency context %q in %s", want, got)
		}
	}
}
