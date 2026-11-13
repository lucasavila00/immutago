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
		if c.statementContext(expression, parent) {
			return
		}
	}
	c.report(expression, "protected operation is outside the supported statement forms")
}

// statementContext checks the first statement or declaration around a protected operand.
func (c *subsetChecks) statementContext(expression ast.Expr, parent ast.Node) bool {
	switch node := parent.(type) {
	case *ast.AssignStmt:
		if len(node.Lhs) != 1 || len(node.Rhs) != 1 {
			if !c.freshGetAssignment(node) {
				c.report(node, "multiple assignments involving protected storage are unsupported")
			}
		}
		return true
	case *ast.ValueSpec:
		if len(node.Names) != 1 || len(node.Values) > 1 {
			if !c.getDeclaration(node) {
				c.report(node, "multiple assignments involving protected storage are unsupported")
			}
		}
		return true
	case *ast.DeclStmt, *ast.IncDecStmt, *ast.ReturnStmt,
		*ast.IfStmt, *ast.ForStmt, *ast.SwitchStmt,
		*ast.FuncDecl, *ast.FuncLit:
		return true
	case *ast.RangeStmt:
		c.rangeTarget(expression, node)
		return true
	case *ast.ExprStmt:
		if _, call := ast.Unparen(node.X).(*ast.CallExpr); call {
			return true
		}
	case *ast.CaseClause:
		if _, ordinary := c.parents[c.parents[node]].(*ast.SwitchStmt); ordinary {
			return true
		}
	default:
		if _, statement := parent.(ast.Stmt); !statement {
			return false
		}
	}
	c.report(expression, "protected operation is outside the supported statement forms")
	return true
}

// rangeTarget requires one existing target when its evaluation uses protected storage.
func (c *subsetChecks) rangeTarget(expression ast.Expr, statement *ast.RangeStmt) {
	if statement.Tok != token.ASSIGN || statement.Key == nil || statement.Value == nil {
		return
	}
	for node := ast.Node(expression); node != statement; node = c.parents[node] {
		if node == statement.Key || node == statement.Value {
			c.report(statement, "multiple assignments involving protected storage are unsupported")
			return
		}
	}
}
