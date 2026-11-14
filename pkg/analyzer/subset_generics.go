package analyzer

import (
	"go/ast"

	"github.com/lucasavila00/immutago/internal/checker"
)

// genericInstance checks one source instance outside unchecked dependency type arguments.
func (c *subsetChecks) genericInstance(name *ast.Ident) {
	instance, found := c.pass.TypesInfo.Instances[name]
	if !found {
		return
	}
	object := c.pass.TypesInfo.Uses[name]
	if libraryObject(object) || object != nil && object.Pkg() != nil && object.Pkg().Path() == checker.RuntimePath {
		return
	}
	for i := 0; i < instance.TypeArgs.Len(); i++ {
		arg := instance.TypeArgs.At(i)
		if c.queries.Contains(arg) || c.slotRelevant(arg) || c.queries.Callable(arg) {
			c.report(name, "user generic instantiations with immutable types are unsupported")
		}
	}
}

// genericStorage rejects user generic declarations that directly use protected storage.
func (c *subsetChecks) genericStorage(declaration ast.Node, name *ast.Ident) {
	ast.Inspect(declaration, func(node ast.Node) bool {
		expression, ok := node.(ast.Expr)
		if !ok {
			return true
		}
		if checker.Unevaluated(c.pass.TypesInfo, c.parents, c.unevaluated, expression) {
			return false
		}
		typ := c.pass.TypesInfo.TypeOf(expression)
		if c.queries.Contains(typ) || c.queries.Callable(typ) {
			c.report(name, "user generic declarations with immutable storage are unsupported")
			return false
		}
		return true
	})
}
