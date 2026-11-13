package analyzer

import (
	"go/ast"

	"github.com/lucasavila00/immutago/internal/checker"
)

// genericStorage rejects user generic declarations that directly use protected storage.
func (c *subsetChecks) genericStorage(declaration ast.Node, name *ast.Ident) {
	ast.Inspect(declaration, func(node ast.Node) bool {
		expression, ok := node.(ast.Expr)
		if !ok {
			return true
		}
		typ := c.pass.TypesInfo.TypeOf(expression)
		if checker.Contains(typ) || protectedCallable(typ) {
			c.report(name, "user generic declarations with immutable storage are unsupported")
			return false
		}
		return true
	})
}
