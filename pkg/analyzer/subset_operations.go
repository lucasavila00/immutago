package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// localReference identifies a pointer to a concrete im.LocalRef struct.
func localReference(t types.Type) bool {
	if t == nil {
		return false
	}
	pointer, ok := t.Underlying().(*types.Pointer)
	return ok && checker.LocalRef(pointer.Elem())
}

// referenceUse checks the concrete target, source, and context of each immutable reference.
func (c *subsetChecks) referenceUse(expression ast.Expr) {
	if name, ok := expression.(*ast.Ident); ok {
		if field, ok := c.pass.TypesInfo.ObjectOf(name).(*types.Var); ok && field.IsField() {
			return
		}
	}
	if c.pass.TypesInfo.Types[expression].IsType() || !protectedReference(c.pass.TypesInfo.TypeOf(expression)) {
		return
	}
	pointer := c.pass.TypesInfo.TypeOf(expression).Underlying().(*types.Pointer)
	if !checker.Marked(pointer.Elem()) {
		c.report(expression, "immutable references require a concrete marked struct target")
		return
	}
	if name, ok := expression.(*ast.Ident); ok && c.pass.TypesInfo.Defs[name] != nil {
		return
	}
	if !c.referenceSource(expression) || !c.referenceContext(expression) {
		c.report(expression, "reference operation is outside the supported allowlist")
	}
}

// referenceSource lists fresh allocations and typed reads of existing immutable references.
func (c *subsetChecks) referenceSource(expression ast.Expr) bool {
	switch source := expression.(type) {
	case *ast.Ident:
		if c.nilExpression(source) {
			return true
		}
		_, variable := c.pass.TypesInfo.ObjectOf(source).(*types.Var)
		return variable
	case *ast.ParenExpr, *ast.SelectorExpr, *ast.IndexExpr:
		return true
	case *ast.UnaryExpr:
		return source.Op == token.AND
	case *ast.CallExpr:
		if builtin, ok := c.calledObject(source.Fun).(*types.Builtin); ok {
			return builtin.Name() == "new"
		}
		_, direct := c.calledObject(source.Fun).(*types.Func)
		return direct
	}
	return false
}

// nilExpression identifies the predeclared nil value, including parentheses.
func (c *subsetChecks) nilExpression(expression ast.Expr) bool {
	name, ok := ast.Unparen(expression).(*ast.Ident)
	return ok && c.pass.TypesInfo.ObjectOf(name) == types.Universe.Lookup("nil")
}

// referenceContext separates stored Shared pointers from local LocalRef pointers.
func (c *subsetChecks) referenceContext(expression ast.Expr) bool {
	local := localReference(c.pass.TypesInfo.TypeOf(expression))
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
		if len(parent.Lhs) != 1 || len(parent.Rhs) != 1 {
			return !local && c.freshGetAssignment(parent)
		}
		if !local {
			return true
		}
		return c.localReferenceBinding(parent.Lhs[0])
	case *ast.ValueSpec:
		return !local || len(parent.Names) == 1 && c.pass.TypesInfo.Defs[parent.Names[0]].Parent() != c.pass.Pkg.Scope()
	case *ast.ReturnStmt, *ast.CompositeLit, *ast.KeyValueExpr:
		return !local
	case *ast.IndexExpr:
		return !local && parent.Index == expression
	case *ast.CallExpr:
		return c.directReferenceArgument(parent, expression)
	}
	return false
}

// localReferenceBinding permits local pointer replacement and explicit discard.
func (c *subsetChecks) localReferenceBinding(destination ast.Expr) bool {
	name, ok := ast.Unparen(destination).(*ast.Ident)
	if !ok {
		return false
	}
	if name.Name == "_" {
		return true
	}
	variable, ok := c.pass.TypesInfo.ObjectOf(name).(*types.Var)
	return ok && variable.Parent() != c.pass.Pkg.Scope()
}

// directReferenceArgument accepts concrete named calls and the fixed shared-reference storage operations.
func (c *subsetChecks) directReferenceArgument(call *ast.CallExpr, expression ast.Expr) bool {
	local := localReference(c.pass.TypesInfo.TypeOf(expression))
	object := c.calledObject(call.Fun)
	if !local && c.ownedBuiltin(call) {
		return true
	}
	function, ok := object.(*types.Func)
	if !ok {
		return false
	}
	if libraryObject(function) {
		selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
		return !local && ok && c.exactCollectionMethod(selector) && function.Name() == "Get"
	}
	if genericSignature(function.Origin().Type().(*types.Signature)) {
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
	signature := c.pass.TypesInfo.TypeOf(call.Fun).Underlying().(*types.Signature)
	if signature.Recv() != nil {
		return false
	}
	for index, argument := range call.Args {
		if argument == expression && index < signature.Params().Len() {
			parameter := signature.Params().At(index).Type()
			return protectedReference(parameter) && types.Identical(parameter, c.pass.TypesInfo.TypeOf(expression))
		}
	}
	return false
}
