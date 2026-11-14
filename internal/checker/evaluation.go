package checker

import (
	"go/ast"
	"go/types"
)

// Unevaluated identifies operands of compiler-proved constant expressions.
// The cache belongs to one pass with fixed type information and parent links.
// Runtime operation and ownership checks use the same evaluation boundary.
func Unevaluated(info *types.Info, parents map[ast.Node]ast.Node, cache map[ast.Node]bool, node ast.Node) bool {
	result := false
	boundary := node
search:
	for ; boundary != nil; boundary = parents[boundary] {
		if cached, found := cache[boundary]; found {
			result = cached
			break
		}
		if expression, ok := boundary.(ast.Expr); ok && info.Types[expression].Value != nil {
			result = true
			break
		}
		switch boundary.(type) {
		case ast.Stmt, *ast.ValueSpec, *ast.TypeSpec:
			break search
		}
	}
	for node != boundary {
		cache[node] = result
		node = parents[node]
	}
	if boundary != nil {
		cache[boundary] = result
	}
	return result
}
