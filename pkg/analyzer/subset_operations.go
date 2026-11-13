package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"
)

// referenceUse accepts only the implemented producers and uses of a protected pointer.
func (c *subsetChecks) referenceUse(expression ast.Expr) {
	info := c.pass.TypesInfo.Types[expression]
	if info.IsType() || !protectedReference(c.pass.TypesInfo.TypeOf(expression)) {
		return
	}
	if id, ok := expression.(*ast.Ident); ok && c.pass.TypesInfo.Defs[id] != nil {
		if protectedVariable(c.pass.TypesInfo.Defs[id]) && !c.referenceDefinition(id) {
			c.report(id, "references require one fixed declaration from an owned local")
		}
		return
	}
	if !c.referenceSource(expression) || !c.referenceContext(expression) {
		c.report(expression, "reference operation is outside the supported allowlist")
	}
}

// referenceSource lists the only expressions that can produce a local protected reference.
func (c *subsetChecks) referenceSource(expression ast.Expr) bool {
	switch source := expression.(type) {
	case *ast.Ident:
		if source.Name == "nil" {
			return true
		}
		variable, ok := c.pass.TypesInfo.ObjectOf(source).(*types.Var)
		return ok && variable.Parent() != c.pass.Pkg.Scope()
	case *ast.ParenExpr:
		return true
	case *ast.UnaryExpr:
		return source.Op == token.AND
	}
	return false
}

// referenceContext lists permitted reference uses; every other Go expression context is rejected.
func (c *subsetChecks) referenceContext(expression ast.Expr) bool {
	switch parent := c.parents[expression].(type) {
	case *ast.ParenExpr:
		return true
	case *ast.StarExpr:
		return parent.X == expression && !c.pass.TypesInfo.Types[parent].IsType()
	case *ast.SelectorExpr:
		return parent.X == expression
	case *ast.BinaryExpr:
		return parent.Op == token.EQL || parent.Op == token.NEQ
	case *ast.AssignStmt:
		if len(parent.Lhs) != 1 || len(parent.Rhs) != 1 || parent.Rhs[0] != expression {
			return false
		}
		name, ok := parent.Lhs[0].(*ast.Ident)
		if !ok {
			return false
		}
		if name.Name == "_" {
			return true
		}
		return parent.Tok == token.DEFINE && c.pass.TypesInfo.Defs[name] != nil && ownerAddress(expression)
	case *ast.ValueSpec:
		return len(parent.Names) == 1 && len(parent.Values) == 1 && parent.Values[0] == expression && ownerAddress(expression)

	case *ast.CallExpr:
		return c.directReferenceArgument(parent, expression)
	}
	return false
}

// directReferenceArgument permits an exact typed parameter on a synchronous, non-generic direct call.
func (c *subsetChecks) directReferenceArgument(call *ast.CallExpr, expression ast.Expr) bool {
	function, ok := c.calledObject(call.Fun).(*types.Func)
	if !ok || libraryObject(function) || genericSignature(function.Origin().Type().(*types.Signature)) {
		return false
	}
	switch c.parents[call].(type) {
	case *ast.GoStmt, *ast.DeferStmt:
		return false
	}
	if selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr); ok {
		if selection := c.pass.TypesInfo.Selections[selector]; selection != nil {
			if _, dynamic := selection.Recv().Underlying().(*types.Interface); dynamic {
				return false
			}
		}
	}
	signature, ok := c.pass.TypesInfo.TypeOf(call.Fun).Underlying().(*types.Signature)
	if !ok {
		return false
	}
	for i, arg := range call.Args {
		if arg != expression || i >= signature.Params().Len() {
			continue
		}
		parameter := signature.Params().At(i).Type()
		return protectedReference(parameter) && types.Identical(parameter, c.pass.TypesInfo.TypeOf(expression))
	}
	return false
}
