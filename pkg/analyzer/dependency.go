package analyzer

import (
	"go/ast"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// uncheckedTypeArguments identifies type syntax whose payload is explicitly trusted.
// It does not exempt the value expression passed to Trust or the code that creates it.
func uncheckedTypeArguments(info *types.Info, expression ast.Expr) bool {
	if info.Types[expression].IsType() && checker.IsRuntimeType(info.TypeOf(expression), checker.UncheckedDependencyType) {
		return true
	}
	var base ast.Expr
	switch expression := expression.(type) {
	case *ast.IndexExpr:
		base = ast.Unparen(expression.X)
	case *ast.IndexListExpr:
		base = ast.Unparen(expression.X)
	default:
		return false
	}
	switch base := base.(type) {
	case *ast.Ident:
		return runtimeObject(info.ObjectOf(base), checker.TrustFunction)
	case *ast.SelectorExpr:
		return runtimeObject(info.ObjectOf(base.Sel), checker.TrustFunction)
	}
	return false
}

// dependencyArgument identifies the exact wrapper or a marked argument struct with a dependency field.
func dependencyArgument(t types.Type) bool {
	if checker.IsRuntimeType(t, checker.UncheckedDependencyType) {
		return true
	}
	structure, ok := t.Underlying().(*types.Struct)
	if !ok || checker.MarkerCount(t) != 1 {
		return false
	}
	for field := range structure.Fields() {
		if checker.TrustedField(field) {
			return true
		}
	}
	return false
}

// trustedValue matches the complete value operand of a direct Trust call.
func (c *subsetChecks) trustedValue(expression ast.Expr) bool {
	call, ok := c.unparenthesizedParent(expression).(*ast.CallExpr)
	return ok && len(call.Args) == 1 && ast.Unparen(call.Args[0]) == ast.Unparen(expression) &&
		runtimeObject(c.calledObject(call.Fun), checker.TrustFunction)
}

// trustedFunctionType permits a function literal signature at the explicit Trust boundary.
func (c *subsetChecks) trustedFunctionType(expression ast.Expr) bool {
	literal, ok := c.parents[expression].(*ast.FuncLit)
	return ok && literal.Type == expression && c.trustedValue(literal)
}
