package checker

import (
	"go/ast"
	"go/types"
)

// Unevaluated identifies operands of compiler-proved constant expressions.
// The runtime operation and ownership checks use the same evaluation boundary.
func Unevaluated(info *types.Info, parents map[ast.Node]ast.Node, node ast.Node) bool {
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
