package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// selectLiteral identifies the sole supported projection callback position.
func (c *subsetChecks) selectLiteral(literal *ast.FuncLit) *ast.CallExpr {
	parent := c.unparenthesizedParent(literal)
	call, ok := parent.(*ast.CallExpr)
	if !ok || len(call.Args) != 2 || ast.Unparen(call.Args[1]) != literal {
		return nil
	}
	object := c.calledObject(call.Fun)
	if libraryObject(object) && object.Name() == "Select" {
		return call
	}
	return nil
}

// matchSelect proves the complete literal and read path, not arbitrary callback effects.
func (c *subsetChecks) matchSelect(call *ast.CallExpr) string {
	const rejected = "Select requires one inline projection path from its parameter"
	if len(call.Args) != 2 || call.Ellipsis.IsValid() {
		return rejected
	}
	if !checker.IsLibraryType(c.pass.TypesInfo.TypeOf(call.Args[0]), "View") {
		return rejected
	}
	literal, ok := ast.Unparen(call.Args[1]).(*ast.FuncLit)
	if !ok || literal.Type.TypeParams != nil || len(literal.Type.Params.List) != 1 ||
		len(literal.Type.Params.List[0].Names) != 1 || literal.Type.Results == nil ||
		len(literal.Type.Results.List) != 1 || len(literal.Type.Results.List[0].Names) != 0 ||
		len(literal.Body.List) != 1 {
		return rejected
	}
	name := literal.Type.Params.List[0].Names[0]
	if name.Name == "_" {
		return rejected
	}
	parameter, ok := c.pass.TypesInfo.Defs[name].(*types.Var)
	if !ok {
		return rejected
	}
	statement, ok := literal.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(statement.Results) != 1 {
		return rejected
	}
	selected := ast.Unparen(statement.Results[0])
	if _, identity := selected.(*ast.Ident); identity || !c.projectionPath(selected, parameter) {
		return rejected
	}
	if reason := c.proofs.FrozenReason(parameter.Type()); reason != "" {
		return reason
	}
	return c.proofs.FrozenReason(c.pass.TypesInfo.TypeOf(selected))
}

// projectionPath accepts direct fields, pointer reads, and fixed-array elements only.
func (c *subsetChecks) projectionPath(expression ast.Expr, parameter *types.Var) bool {
	switch value := ast.Unparen(expression).(type) {
	case *ast.Ident:
		return c.pass.TypesInfo.ObjectOf(value) == parameter
	case *ast.SelectorExpr:
		selection := c.pass.TypesInfo.Selections[value]
		return selection != nil && selection.Kind() == types.FieldVal && len(selection.Index()) == 1 &&
			c.projectionPath(value.X, parameter)
	case *ast.StarExpr:
		pointer, ok := c.pass.TypesInfo.TypeOf(value.X).Underlying().(*types.Pointer)
		return ok && checker.Frozen(pointer.Elem()) && c.projectionPath(value.X, parameter)
	case *ast.IndexExpr:
		_, array := c.pass.TypesInfo.TypeOf(value.X).Underlying().(*types.Array)
		return array && c.projectionPath(value.X, parameter) && c.projectionIndex(value.Index)
	}
	return false
}

// projectionIndex accepts integer constants and ordinary integer local bindings.
func (c *subsetChecks) projectionIndex(expression ast.Expr) bool {
	expression = ast.Unparen(expression)
	basic, ok := c.pass.TypesInfo.TypeOf(expression).Underlying().(*types.Basic)
	if !ok || basic.Info()&types.IsInteger == 0 {
		return false
	}
	switch value := expression.(type) {
	case *ast.BasicLit:
		return value.Kind == token.INT || value.Kind == token.CHAR
	case *ast.Ident:
		switch object := c.pass.TypesInfo.ObjectOf(value).(type) {
		case *types.Const:
			return true
		case *types.Var:
			return !object.IsField() && object.Parent() != object.Pkg().Scope()
		}
	case *ast.SelectorExpr:
		_, constant := c.pass.TypesInfo.ObjectOf(value.Sel).(*types.Const)
		return constant && c.pass.TypesInfo.Selections[value] == nil
	}
	return false
}
