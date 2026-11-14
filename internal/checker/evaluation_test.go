package checker

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

// evaluationWalk is the independent ancestor-walk oracle without cached results.
func evaluationWalk(info *types.Info, parents map[ast.Node]ast.Node, node ast.Node) bool {
	for ; node != nil; node = parents[node] {
		if expression, ok := node.(ast.Expr); ok && info.Types[expression].Value != nil {
			return true
		}
		switch node.(type) {
		case ast.Stmt, *ast.ValueSpec, *ast.TypeSpec:
			return false
		}
	}
	return false
}

// evaluationSource provides constant operands, runtime calls, and nested statement boundaries.
func evaluationSource(t *testing.T) (*types.Info, map[ast.Node]ast.Node, []ast.Node) {
	t.Helper()
	const source = `package sample
const count = len([3]int{})
type Row [len([2]int{})]int
func read() int { return 1 }
func run() int {
 x := read()
 _ = len([2]int{x, x})
 _ = len([2]int{read(), 0})
 _ = len([1]func(){func(){ x := read(); _ = x }})
 if len([1]int{}) > 0 { x++ }
 var row Row
 _ = row
 return x
}`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "evaluation.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue)}
	if _, err := new(types.Config).Check("sample", fset, []*ast.File{file}, info); err != nil {
		t.Fatal(err)
	}
	parents := make(map[ast.Node]ast.Node)
	var nodes, stack []ast.Node
	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return false
		}
		if len(stack) > 0 {
			parents[node] = stack[len(stack)-1]
		}
		nodes = append(nodes, node)
		stack = append(stack, node)
		return true
	})
	return info, parents, nodes
}

// TestEvaluationCacheOrder compares every AST node with the oracle in different query orders.
func TestEvaluationCacheOrder(t *testing.T) {
	info, parents, nodes := evaluationSource(t)
	for _, reverse := range []bool{false, true} {
		cache := make(map[ast.Node]bool)
		for pass := 0; pass < 2; pass++ {
			for index := range nodes {
				query := index
				if reverse {
					query = len(nodes) - 1 - index
				}
				node := nodes[query]
				if got, want := Unevaluated(info, parents, cache, node), evaluationWalk(info, parents, node); got != want {
					t.Fatalf("reverse=%t pass=%d node=%T at %d: got %t, want %t", reverse, pass, node, node.Pos(), got, want)
				}
			}
		}
		if Unevaluated(info, parents, cache, nil) {
			t.Fatal("nil node is unevaluated")
		}
	}
}

// TestEvaluationCacheBoundaries checks a constant outer expression cannot cross a function body statement.
func TestEvaluationCacheBoundaries(t *testing.T) {
	info, parents, nodes := evaluationSource(t)
	cache := make(map[ast.Node]bool)
	constants, runtime := 0, 0
	for _, node := range nodes {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			continue
		}
		name, ok := call.Fun.(*ast.Ident)
		if !ok {
			continue
		}
		if name.Name == "read" {
			runtime++
			if Unevaluated(info, parents, cache, call) {
				t.Fatal("runtime read call became unevaluated")
			}
		}
		if name.Name == "len" && info.Types[call].Value != nil {
			constants++
			if !Unevaluated(info, parents, cache, call.Args[0]) {
				t.Fatal("constant len operand became evaluated")
			}
		}
	}
	if constants < 4 || runtime != 3 {
		t.Fatalf("fixture coverage: %d constants, %d runtime calls", constants, runtime)
	}
}
