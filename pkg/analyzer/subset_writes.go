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
		if !c.protectedSelection(target) && !c.protectedRoot(target.X) &&
			!c.queries.Stored(c.pass.TypesInfo.TypeOf(target)) {
			return ""
		}
	case *ast.IndexExpr:
		if c.ownedExpression(target.X) || (!c.protectedRoot(target.X) &&
			!c.queries.Stored(c.pass.TypesInfo.TypeOf(target))) {
			return ""
		}
	}
	if !c.protectedRoot(destination) {
		return ""
	}
	return "writes to immutable objects are unsupported"
}
