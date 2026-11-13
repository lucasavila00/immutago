package borrow

import (
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

// TestLocalLoans exercises whole-owner liveness, alias copies, rebinding, and readonly permissions.
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
		{"alias copy", `var v Value;p:=&v;q:=p;v.X=2;read(q)`, true},
		{"rebind", `var a,b Value;p:=&a;p=&b;a.X=2;read(p)`, false},
		{"branch", `var a,b Value;p:=&a;if flag {p=&b};a.X=2;read(p)`, true},
		{"readonly", `var v Value;p:=&v;p.X=2`, true},
		{"fresh readonly", `p:=&Value{};p.X=2`, true},
		{"dead simultaneous borrow", `var v Value;p:=&v;read(p);v,p=Value{},&v`, false},
		{"simultaneous borrow", `var v Value;var p *Value;v,p=Value{},&v;read(p)`, true},
		{"simultaneous copied borrow", `var v Value;p:=&v;var q *Value;v,q=Value{},p;read(q)`, true},
		{"swap", `var a,b Value;p,q:=&a,&b;p,q=q,p;a.X=2;read(q);read(p)`, true},
		{"loop carried", `var v Value;var old *Value;for i:=0;i<2;i++{v.X=i;p:=&v;if old!=nil{read(old)};old=p}`, true},
		{"loop last use", `var v Value;var old *Value;for i:=0;i<2;i++{if old!=nil{read(old)};v.X=i;old=&v}`, false},
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
