package analyzer

import (
	"github.com/lucasavila00/immutago/internal/checker"
	"go/ast"
)

// matchWrite permits variable replacement and whole native builder slots.
func (c *subsetChecks) matchWrite(destination ast.Expr) checker.Failure {
	switch target := ast.Unparen(destination).(type) {
	case *ast.Ident:
		return checker.Failure{}
	case *ast.SelectorExpr:
		if c.pass.TypesInfo.Selections[target] == nil {
			return checker.Failure{}
		}
		if !c.protectedSelection(target) && !c.protectedRoot(target.X) &&
			!c.queries.Owns(c.pass.TypesInfo.TypeOf(target)) {
			return checker.Failure{}
		}
	case *ast.IndexExpr:
		if c.ownedExpression(target.X) || (!c.protectedRoot(target.X) &&
			!c.queries.Owns(c.pass.TypesInfo.TypeOf(target))) {
			return checker.Failure{}
		}
	}
	if !c.protectedRoot(destination) {
		return checker.Failure{}
	}
	return shapeFailure("writes to immutable objects are unsupported")
}
