package borrow

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

// localImporter gives local tests the sealed marker's exact compiler package identity.
type localImporter struct{ library *types.Package }

// Import resolves the sole library import in local analysis fixtures.
func (i localImporter) Import(string) (*types.Package, error) { return i.library, nil }

// TestLocalLoans checks fixed-owner references, whole-owner lifetimes, and read-only access.
func TestLocalLoans(t *testing.T) {
	cases := []struct {
		name, body string
		rejected   bool
	}{
		{"construction", `var v Value; if flag {v.X=1}else{v.X=2}; read(&v)`, false},
		{"last use", `var v Value;p:=&v;read(p);v.X=2`, false},
		{"live alias", `var v Value;p:=&v;v.X=2;read(p)`, true},
		{"whole owner", `var v Value;p:=&v;v.Y=2;read(p)`, true},
		{"rhs use", `var v Value;p:=&v;v.X=read(p)+1`, false},
		{"branch", `var v Value;p:=&v;if flag {v.X=2};read(p)`, true},
		{"two borrows", `var v Value;p:=&v;q:=&v;read(p);v.X=2;read(q)`, true},
		{"branch last use", `var v Value;p:=&v;if flag {read(p)}else{read(p)};v.X=2`, false},
		{"loop live", `var v Value;p:=&v;for i:=0;i<2;i++{v.X=i;read(p)}`, true},
		{"loop local last use", `var v Value;for i:=0;i<2;i++{p:=&v;read(p);v.X=i}`, false},
		{"typed declaration", `var v Value;var p *Value=&v;v.X=2;read(p)`, true},
		{"readonly", `var v Value;p:=&v;p.X=2`, true},
		{"fresh readonly", `p:=&Value{};p.X=2`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			diagnostics := localDiagnostics(t, tc.body)
			if (diagnostics != "") != tc.rejected {
				t.Fatalf("diagnostics=%q rejected=%v", diagnostics, tc.rejected)
			}
		})
	}
}

// localDiagnostics compiles a fixture and runs only the local borrow proof.
func localDiagnostics(t *testing.T, body string, declarations ...string) string {
	t.Helper()
	positions := token.NewFileSet()
	libraryFile, err := parser.ParseFile(positions, "im.go", "package im;type Marker struct{}", 0)
	if err != nil {
		t.Fatal(err)
	}
	library, err := new(types.Config).Check(checker.LibraryPath, positions, []*ast.File{libraryFile}, nil)
	if err != nil {
		t.Fatal(err)
	}
	source := `package sample;import "` + checker.LibraryPath + `";type Value struct{im.Marker;X,Y int};
 func read(p *Value)int{return p.X};func test(flag bool){` + body + `}` + "\n" + strings.Join(declarations, "\n")
	file, err := parser.ParseFile(positions, "sample.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue), Defs: make(map[*ast.Ident]types.Object),
		Uses: make(map[*ast.Ident]types.Object), Selections: make(map[*ast.SelectorExpr]*types.Selection)}
	config := types.Config{Importer: localImporter{library}}
	pkg, err := config.Check("sample", positions, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	var reports []string
	pass := &analysis.Pass{Fset: positions, Pkg: pkg, Files: []*ast.File{file}, TypesInfo: info,
		Report: func(d analysis.Diagnostic) { reports = append(reports, d.Message) }}
	if err := Check(pass); err != nil {
		t.Fatal(err)
	}
	return strings.Join(reports, "\n")
}

// TestLocalGlobalConstructionBoundary permits declaration initialization but forbids updates in init bodies.
func TestLocalGlobalConstructionBoundary(t *testing.T) {
	if got := localDiagnostics(t, "", `var Global = Value{X:1}`); got != "" {
		t.Fatal(got)
	}
	got := localDiagnostics(t, "", `var Global Value; func init(){Global.X=1}`)
	if !strings.Contains(got, "cannot write package-level immutable storage") {
		t.Fatalf("initializer diagnostics = %q", got)
	}
}

// TestLocalReaderParameters checks that parameters need no owner entry to enforce read-only access.
func TestLocalReaderParameters(t *testing.T) {
	got := localDiagnostics(t, "", `func change(p *Value){p.X=2}`)
	if !strings.Contains(got, "cannot write through shared reference") {
		t.Fatal(got)
	}
}

// TestLocalSeparateScopes checks many short loans without a per-point map of possible owners.
func TestLocalSeparateScopes(t *testing.T) {
	for _, count := range []int{50, 100, 200} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			body := strings.Repeat(`{var v Value;p:=&v;read(p);v.X=2};`, count)
			if got := localDiagnostics(t, body); got != "" {
				t.Fatal(got)
			}
		})
	}
}
