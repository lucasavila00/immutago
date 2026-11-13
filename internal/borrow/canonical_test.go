package borrow

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
)

// TestRecursiveLocalTraversalConverges exercises the former unbounded p.Next provenance expansion.
func TestRecursiveLocalTraversalConverges(t *testing.T) {
	source := `package sample
 type Node struct { Next *Node }
 func walk(p *Node) { for p != nil { p = p.Next } }
 type Chain map[int]Chain
 func descend(p Chain) { for p != nil { p = p[0] } }
 `
	positions := token.NewFileSet()
	file, err := parser.ParseFile(positions, "loop.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue), Defs: make(map[*ast.Ident]types.Object),
		Uses: make(map[*ast.Ident]types.Object), Instances: make(map[*ast.Ident]types.Instance),
		Selections: make(map[*ast.SelectorExpr]*types.Selection), Scopes: make(map[ast.Node]*types.Scope),
		Implicits: make(map[ast.Node]types.Object)}
	config := types.Config{}
	pkg, err := config.Check("sample", positions, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	pass := &analysis.Pass{Fset: positions, Files: []*ast.File{file}, Pkg: pkg, TypesInfo: info,
		TypesSizes: types.SizesFor("gc", "amd64"),
		Report:     func(d analysis.Diagnostic) { t.Errorf("unexpected: %s", d.Message) }}
	if err := Check(pass); err != nil {
		t.Fatal(err)
	}
}

// TestOpaqueProjectionCannotKillLoans checks absorption and conservative overlap independently of a CFG.
func TestOpaqueProjectionCannotKillLoans(t *testing.T) {
	opaque := Place{Root: 1, Path: []Projection{{Kind: 'f', Index: 0}, {Kind: '*'}}}
	root := Place{Root: 1}
	if !Overlap(root, opaque, false) || Overlap(root, opaque, true) {
		t.Fatal("opaque projection used as a definite kill")
	}
	if opaque.project('f', 2).key() != opaque.key() {
		t.Fatal("opaque tail did not absorb a further projection")
	}
	if definitePlace(opaque) {
		t.Fatal("opaque cell permits a strong memory update")
	}
}
