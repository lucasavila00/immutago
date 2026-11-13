package freeze

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

const librarySource = `package im
 type Marker struct{}
 type FrozenSlice[T any] struct{}
 type FrozenMap[K comparable,V any] struct{}
 func FreezeSlice[T any](v []T)FrozenSlice[T]{return FrozenSlice[T]{}}
 func FreezeMap[K comparable,V any](v map[K]V)FrozenMap[K,V]{return FrozenMap[K,V]{}}
 func(v FrozenSlice[T])Clone()[]T{return nil}
 func(v FrozenMap[K,V])Clone()map[K]V{return nil}
`

// libraryImporter resolves the sealed library identities for compiler-driven ownership tests.
type libraryImporter struct{ library *types.Package }

// Import supplies the test library without loading implementation dependencies.
func (i libraryImporter) Import(string) (*types.Package, error) { return i.library, nil }

// TestFreezeOwnership exercises fresh ownership, forbidden aliases, and control-flow consumption.
func TestFreezeOwnership(t *testing.T) {
	cases := []struct {
		name, body string
		reject     bool
	}{
		{"slice builder", `s:=make([]int,1);s[0]=2;s=append(s,3);_=im.FreezeSlice(s)`, false},
		{"map builder", `m:=map[string]int{};m["x"]=1;delete(m,"x");_=im.FreezeMap(m)`, false},
		{"ordinary scalar clone", `f:=im.FreezeSlice([]int{1});s:=f.Clone();take(s)`, false},
		{"clone escape before refreeze", `f:=im.FreezeSlice([]int{1});s:=f.Clone();take(s);_=im.FreezeSlice(s)`, true},
		{"clone", `f:=im.FreezeSlice([]int{1});s:=f.Clone();s[0]=2;_=im.FreezeSlice(s)`, false},
		{"borrowed slice clone", `f:=im.FreezeSlice([]int{1});p:=&f;s:=p.Clone();s[0]=2;_=im.FreezeSlice(s)`, false},
		{"dereferenced slice clone", `f:=im.FreezeSlice([]int{1});p:=&f;s:=(*p).Clone();s[0]=2;_=im.FreezeSlice(s)`, false},
		{"borrowed map clone", `f:=im.FreezeMap(map[string]int{"x":1});p:=&f;m:=p.Clone();m["x"]=2;_=im.FreezeMap(m)`, false},
		{"dereferenced map clone", `f:=im.FreezeMap(map[string]int{"x":1});p:=&f;m:=(*p).Clone();` +
			`m["x"]=2;_=im.FreezeMap(m)`, false},
		{"named clone", `v:=Fake{};p:=&v;s:=p.Clone();_=im.FreezeSlice(s)`, true},
		{"nil", `var s []int;_=im.FreezeSlice(s);_=im.FreezeMap[string,int](nil)`, false},
		{"marked builder", `s:=[]V{{X:1}};s[0].X=2;_=s[0]`, false},
		{"loop fresh", `for i:=0;i<2;i++{s:=[]int{i};_=im.FreezeSlice(s)}`, false},
		{"read after", `s:=[]int{1};_=im.FreezeSlice(s);_=s[0]`, true},
		{"write after", `s:=[]int{1};_=im.FreezeSlice(s);s[0]=2`, true},
		{"len after", `s:=[]int{1};_=im.FreezeSlice(s);_=len(s)`, true},
		{"dead alias", `s:=[]int{1};a:=s;_=a;_=im.FreezeSlice(s)`, true},
		{"slice view", `s:=[]int{1};a:=s[:];_=a;_=im.FreezeSlice(s)`, true},
		{"address", `s:=[]int{1};p:=&s[0];_=p;_=im.FreezeSlice(s)`, true},
		{"escape", `s:=[]int{1};take(s);_=im.FreezeSlice(s)`, true},
		{"capture", `s:=[]int{1};g:=func(){_=len(s)};g();_=im.FreezeSlice(s)`, true},
		{"append alias", `s:=[]int{1};a:=append(s,2);_=a;_=im.FreezeSlice(s)`, true},
		{"rebind", `s:=[]int{1};_=im.FreezeSlice(s);s=[]int{2};_=s`, true},
		{"branch", `s:=[]int{1};if flag{_=im.FreezeSlice(s)};_=s[0]`, true},
		{"repeated loop", `s:=[]int{1};for i:=0;i<2;i++{_=im.FreezeSlice(s)}`, true},
		{"active range", `s:=[]int{1};for range s{_=im.FreezeSlice(s)}`, true},
		{"post assignment", `s:=[]int{1};var f im.FrozenSlice[int];s[0],f=2,im.FreezeSlice(s);_=f`, true},
		{"post map assignment", `m:=map[string]int{};var f im.FrozenMap[string,int];m["x"],f=2,im.FreezeMap(m);_=f`, true},
		{"post index", `s:=[]int{1};_=s[index(im.FreezeSlice(s))]`, true},
		{"post increment", `s:=[]int{1};s[index(im.FreezeSlice(s))]++`, true},
		{"post delete", `m:=map[string]int{};delete(m,key(im.FreezeMap(m)))`, true},
		{"post copy", `s:=[]int{1};copy(s,im.FreezeSlice(s).Clone())`, true},
		{"deferred mutation", `s:=[]int{1};defer clear(s);_=im.FreezeSlice(s)`, true},
		{"concurrent mutation", `s:=[]int{1};go clear(s);_=im.FreezeSlice(s)`, true},
		{"array view", `s:=[][1]int{{1}};v:=s[0][:];_=im.FreezeSlice(s);v[0]=2`, true},
		{"argument order", `s:=[]int{1};both(im.FreezeSlice(s),s[0])`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := diagnostics(t, tc.body)
			if (got != "") != tc.reject {
				t.Fatalf("diagnostics=%q rejected=%v", got, tc.reject)
			}
		})
	}
}

// diagnostics type-checks independent source and runs the ownership proof directly.
func diagnostics(t *testing.T, body string) string {
	t.Helper()
	positions := token.NewFileSet()
	libraryFile, err := parser.ParseFile(positions, "im.go", librarySource, 0)
	if err != nil {
		t.Fatal(err)
	}
	library, err := new(types.Config).Check(checker.LibraryPath, positions, []*ast.File{libraryFile}, nil)
	if err != nil {
		t.Fatal(err)
	}
	source := `package sample;import "` + checker.LibraryPath + `";type V struct{im.Marker;X int};
	 type Fake struct{};func(Fake)Clone()[]int{return nil};
 func take([]int){};func both(im.FrozenSlice[int],int){};
 func index(im.FrozenSlice[int])int{return 0};func key(im.FrozenMap[string,int])string{return "x"};
 func test(flag bool){` + body + `}`
	file, err := parser.ParseFile(positions, "sample.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue), Defs: make(map[*ast.Ident]types.Object),
		Uses: make(map[*ast.Ident]types.Object), Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Instances: make(map[*ast.Ident]types.Instance)}
	config := types.Config{Importer: libraryImporter{library}}
	pkg, err := config.Check("sample", positions, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	var reports []string
	pass := &analysis.Pass{Fset: positions, Pkg: pkg, Files: []*ast.File{file}, TypesInfo: info,
		Report: func(d analysis.Diagnostic) { reports = append(reports, d.Message) }}
	Check(pass)
	return strings.Join(reports, "\n")
}
