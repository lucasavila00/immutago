package analyzer

import "go/ast"

// matchWrite permits variable replacement and whole native builder slots.
func (c *subsetChecks) matchWrite(destination ast.Expr) string {
	switch target := ast.Unparen(destination).(type) {
	case *ast.Ident:
		return ""
	case *ast.SelectorExpr:
		if c.pass.TypesInfo.Selections[target] == nil {
			return ""
		}
	case *ast.IndexExpr:
		if c.ownedExpression(target.X) {
			return ""
		}
	}
	if !c.protectedRoot(destination) {
		return ""
	}
	return "writes to immutable objects are unsupported"
}
