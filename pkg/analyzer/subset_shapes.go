package analyzer

import "go/ast"

// statementOperation identifies the statement or declaration that owns an operand.
func (c *subsetChecks) statementOperation(expression ast.Expr) ast.Node {
	for parent := c.parents[expression]; parent != nil; parent = c.parents[parent] {
		switch parent.(type) {
		case ast.Stmt, *ast.ValueSpec:
			return parent
		}
	}
	return expression
}

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

// matchStatement finds the complete statement that owns a protected operand.
func (c *subsetChecks) matchStatement(expression ast.Expr) string {
	for parent := c.parents[expression]; parent != nil; parent = c.parents[parent] {
		if declaration, ok := parent.(*ast.ValueSpec); ok {
			return c.matchDeclaration(declaration)
		}
		statement, ok := parent.(ast.Stmt)
		if !ok {
			continue
		}
		if clause, ok := c.parents[statement].(*ast.CommClause); ok && clause.Comm == statement {
			return "protected storage is unsupported in select communication clauses"
		}
		return c.matchStatementNode(statement)
	}
	return "protected operation is outside the supported statement forms"
}

// matchStatementNode lists each complete supported statement and rejects the remaining Go statements.
func (c *subsetChecks) matchStatementNode(statement ast.Stmt) string {
	switch statement := statement.(type) {
	case *ast.AssignStmt:
		return c.matchAssignment(statement)
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
	}
	return "protected operation is outside the supported statement forms"
}
