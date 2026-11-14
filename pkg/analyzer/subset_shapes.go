package analyzer

import "go/ast"

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

// matchStatement accepts only complete documented statement forms around a protected operation.
func (c *subsetChecks) matchStatement(expression ast.Expr) string {
	for parent := c.parents[expression]; parent != nil; parent = c.parents[parent] {
		if statement, ok := parent.(ast.Stmt); ok {
			if clause, ok := c.parents[statement].(*ast.CommClause); ok && clause.Comm == statement {
				return "protected storage is unsupported in select communication clauses"
			}
		}
		switch statement := parent.(type) {
		case *ast.AssignStmt:
			return c.matchAssignment(statement)
		case *ast.ValueSpec:
			return c.matchDeclaration(statement)
		case *ast.ReturnStmt:
			return c.matchReturn(statement)
		case *ast.IncDecStmt:
			return c.matchWrite(statement.X)
		case *ast.RangeStmt:
			return c.matchRange(statement)
		case *ast.SwitchStmt:
			return c.matchSwitch(statement)
		case *ast.IfStmt, *ast.ForStmt:
			return ""
		case *ast.ExprStmt:
			if call, ok := ast.Unparen(statement.X).(*ast.CallExpr); ok {
				return c.matchCall(call)
			}
		case *ast.GoStmt, *ast.DeferStmt:
			return "protected storage requires synchronous calls"
		case *ast.CaseClause:
			if block, ok := c.parents[statement].(*ast.BlockStmt); ok {
				if parent, ok := c.parents[block].(*ast.SwitchStmt); ok {
					return c.matchSwitch(parent)
				}
			}
		default:
			if _, statement := parent.(ast.Stmt); !statement {
				continue
			}
		}
		return "protected operation is outside the supported statement forms"
	}
	return "protected operation is outside the supported statement forms"
}
