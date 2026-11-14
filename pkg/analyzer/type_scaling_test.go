package analyzer

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
)

// ordinaryTypeGraph builds connected storage and callback types without importing im.
func ordinaryTypeGraph(count int) string {
	var source strings.Builder
	source.WriteString("package ordinary\n")
	for i := range count {
		fmt.Fprintf(&source, "type S%d struct { Next *S%d; Many []S%d; ByKey map[string]*S%d; Svc I%d; N int }\n",
			i, (i+1)%count, (i*7+3)%count, (i*13+5)%count, (i+2)%count)
		fmt.Fprintf(&source, "type I%d interface { Get(string,*S%d)(*S%d,error); Put(S%d,func(*S%d)error)error }\n",
			i, i, (i+4)%count, (i+5)%count, (i+6)%count)
	}
	for i := range count {
		fmt.Fprintf(&source, "func F%d(s *S%d)(int,error) { total:=0;\n", i, i)
		for j := range 10 {
			fmt.Fprintf(&source, "if s.Svc!=nil { v,err:=s.Svc.Get(%q,s.Next.Next); "+
				"if err!=nil {return total,err}; total+=v.N+len(s.Many)+len(s.ByKey) };\n", fmt.Sprint(j))
		}
		source.WriteString("return total,nil }\n")
	}
	return source.String()
}

// ordinaryGraphPass excludes compiler setup from the repeated package-analysis measurements.
func ordinaryGraphPass(t testing.TB, count int) *analysis.Pass {
	t.Helper()
	positions := token.NewFileSet()
	file, err := parser.ParseFile(positions, "ordinary.go", ordinaryTypeGraph(count), 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue), Defs: make(map[*ast.Ident]types.Object),
		Uses: make(map[*ast.Ident]types.Object), Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes: make(map[ast.Node]*types.Scope), Instances: make(map[*ast.Ident]types.Instance)}
	pkg, err := new(types.Config).Check("ordinary", positions, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	return &analysis.Pass{Analyzer: Analyzer, Fset: positions, Pkg: pkg, Files: []*ast.File{file}, TypesInfo: info,
		Report: func(d analysis.Diagnostic) { t.Fatalf("ordinary source rejected: %s", d.Message) }}
}

// TestOrdinaryTypeGraph checks that ordinary recursive interfaces remain outside protected operations.
func TestOrdinaryTypeGraph(t *testing.T) {
	if _, err := run(ordinaryGraphPass(t, 8)); err != nil {
		t.Fatal(err)
	}
}

// BenchmarkCoreOrdinary measures repeated queries in the complete analyzer at increasing graph sizes.
func BenchmarkCoreOrdinary(b *testing.B) {
	for _, count := range []int{8, 16, 32, 64} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			pass := ordinaryGraphPass(b, count)
			b.ReportAllocs()
			for b.Loop() {
				if _, err := run(pass); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
