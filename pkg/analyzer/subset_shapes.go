package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"
)

// protectedStorage includes native builders and scalar projections of protected owners.
func (c *subsetChecks) protectedStorage(expression ast.Expr) bool {
	if c.protectedRoot(expression) || c.ownedExpression(expression) {
		return true
	}
	switch node := ast.Unparen(expression).(type) {
	case *ast.IndexExpr:
		return c.protectedStorage(node.X)
	case *ast.SelectorExpr:
		return c.protectedStorage(node.X)
	case *ast.StarExpr:
		return c.protectedStorage(node.X)
	}
	return false
}

// statementShape permits only documented statement forms for protected expressions.
func (c *subsetChecks) statementShape(expression ast.Expr) {
	if name, ok := expression.(*ast.Ident); ok {
		if _, typeName := c.pass.TypesInfo.ObjectOf(name).(*types.TypeName); typeName {
			return
		}
	}

	if c.pass.TypesInfo.Types[expression].IsType() ||
		(!c.protectedStorage(expression) && !protectedResult(c.pass.TypesInfo.TypeOf(expression))) {
		return
	}
	if name, ok := expression.(*ast.Ident); ok && c.pass.TypesInfo.Defs[name] != nil {
		return
	}
	for parent := c.parents[expression]; parent != nil; parent = c.parents[parent] {
		switch node := parent.(type) {
		case *ast.AssignStmt:
			if len(node.Lhs) == 1 && len(node.Rhs) == 1 || c.freshGetAssignment(node) {
				return
			}
			c.report(node, "multiple assignments involving protected storage are unsupported")
			return
		case *ast.DeclStmt, *ast.IncDecStmt, *ast.ReturnStmt,
			*ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt:
			return
		case *ast.ExprStmt:
			if _, call := ast.Unparen(node.X).(*ast.CallExpr); call {
				return
			}
		case *ast.ValueSpec:
			if len(node.Names) == 1 && len(node.Values) <= 1 || c.getDeclaration(node) {
				return
			}
			c.report(node, "multiple assignments involving protected storage are unsupported")
			return
		case *ast.CaseClause:
			if _, ordinary := c.parents[c.parents[node]].(*ast.SwitchStmt); ordinary {
				return
			}
		case *ast.FuncDecl, *ast.FuncLit:
			return
		default:
			if _, statement := parent.(ast.Stmt); !statement {
				continue
			}
		}
		c.report(expression, "protected operation is outside the supported statement forms")
		return
	}
	c.report(expression, "protected operation is outside the supported statement forms")
}

// referenceDefinition admits only a named reader parameter or one address-initialized local binding.
func (c *subsetChecks) referenceDefinition(name *ast.Ident) bool {
	object := c.pass.TypesInfo.Defs[name]
	if object == nil || object.Parent() == c.pass.Pkg.Scope() {
		return false
	}
	switch parent := c.parents[name].(type) {
	case *ast.AssignStmt:
		return parent.Tok == token.DEFINE && len(parent.Lhs) == 1 && len(parent.Rhs) == 1 && ownerAddress(parent.Rhs[0])
	case *ast.ValueSpec:
		return len(parent.Names) == 1 && len(parent.Values) == 1 && ownerAddress(parent.Values[0])
	case *ast.Field:
		list, ok := c.parents[parent].(*ast.FieldList)
		if !ok {
			return false
		}
		signature, ok := c.parents[list].(*ast.FuncType)
		if !ok || signature.Params != list || signature.TypeParams != nil {
			return false
		}
		declaration, ok := c.parents[signature].(*ast.FuncDecl)
		return ok && declaration.Recv == nil
	}
	return false
}

// protectedVariable identifies reference definitions without treating field metadata as a use.
func protectedVariable(object types.Object) bool {
	variable, ok := object.(*types.Var)
	return ok && !variable.IsField() && protectedReference(variable.Type())
}
