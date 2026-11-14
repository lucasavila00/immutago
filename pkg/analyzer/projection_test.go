package analyzer

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"
	"testing"
)

// BenchmarkCoreStorageProjection measures one ordinary projection classification.
func BenchmarkCoreStorageProjection(b *testing.B) {
	for _, depth := range []int{16, 64, 256} {
		b.Run(fmt.Sprint(depth), func(b *testing.B) {
			source := "var _ = im.Shared{}\nfunc f(a " + strings.Repeat("[1]", depth) +
				"int) int { return a" + strings.Repeat("[0]", depth) + " }"
			pass := sourcePass(b, source)
			var result ast.Expr
			ast.Inspect(pass.Files[0], func(node ast.Node) bool {
				if statement, ok := node.(*ast.ReturnStmt); ok {
					result = statement.Results[0]
				}
				return true
			})
			checks := &subsetChecks{pass: pass}
			if checks.protectedStorage(result) {
				b.Fatal("ordinary projection is protected")
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_ = checks.protectedStorage(result)
			}
		})
	}
}

// TestStorageProjection checks ordinary, marked, and exclusive builder roots.
func TestStorageProjection(t *testing.T) {
	pass := sourcePass(t, `type V struct { im.Shared; A [1][1]int }
type Plain struct { Inner struct { X int } }
func f(p *V, a [1][1]int, v Plain) {
 builder := []int{1}
 _ = ((p)).A[0][0]
 _ = ((*p)).A[0][0]
 _ = ((builder))[0]
 _ = ((a))[0][0]
 _ = ((v)).Inner.X
}`)
	owner, expressions := projectionTestInputs(pass.Files[0], pass.TypesInfo)
	if owner == nil || len(expressions) != 5 {
		t.Fatalf("missing builder or projection expressions: %v, %d", owner, len(expressions))
	}
	checks := &subsetChecks{pass: pass, owners: map[*types.Var]bool{owner: true}}
	for i, expression := range expressions {
		if got, want := checks.protectedStorage(expression), i < 3; got != want {
			t.Errorf("projection %d: got %t, want %t", i, got, want)
		}
	}
}

// projectionTestInputs finds the builder definition and the discarded reads.
func projectionTestInputs(file *ast.File, info *types.Info) (*types.Var, []ast.Expr) {
	var owner *types.Var
	var expressions []ast.Expr
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.Ident:
			if node.Name == "builder" {
				if variable, ok := info.Defs[node].(*types.Var); ok {
					owner = variable
				}
			}
		case *ast.AssignStmt:
			if len(node.Lhs) == 1 && len(node.Rhs) == 1 {
				if name, ok := node.Lhs[0].(*ast.Ident); ok && name.Name == "_" {
					expressions = append(expressions, node.Rhs[0])
				}
			}
		}
		return true
	})
	return owner, expressions
}
