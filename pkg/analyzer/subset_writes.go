package analyzer

import "go/ast"

// write permits variable replacement and native builder elements, but no immutable object mutation.
func (c *subsetChecks) write(destination ast.Expr) {
	switch target := ast.Unparen(destination).(type) {
	case *ast.Ident:
		return
	case *ast.SelectorExpr:
		if c.pass.TypesInfo.Selections[target] == nil {
			return
		}
	case *ast.IndexExpr:
		if c.ownedExpression(target.X) {
			return
		}
	}
	if c.protectedRoot(destination) {
		c.report(destination, "writes to immutable objects are unsupported")
	}
}
