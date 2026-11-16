package analyzer

import (
	"go/ast"
	"go/types"
)

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

// protectedStorage includes native builders and fields or elements of protected owners.
func (c *subsetChecks) protectedStorage(expression ast.Expr) bool {
	for expression != nil {
		if c.pass.TypesInfo.Types[expression].IsType() {
			return false
		}
		if c.queries.Owns(c.pass.TypesInfo.TypeOf(expression)) {
			return true
		}
		switch node := expression.(type) {
		case *ast.Ident:
			return c.ownedExpression(node)
		case *ast.ParenExpr:
			expression = node.X
		case *ast.IndexExpr:
			expression = node.X
		case *ast.SelectorExpr:
			if c.protectedSelection(node) {
				return true
			}
			expression = node.X
		case *ast.StarExpr:
			expression = node.X
		default:
			return false
		}
	}
	return false
}

// protectedSelection checks each owner of a compiler-selected field.
func (c *subsetChecks) protectedSelection(selector *ast.SelectorExpr) bool {
	selection := c.pass.TypesInfo.Selections[selector]
	if selection == nil || selection.Kind() != types.FieldVal {
		return false
	}
	owner := selection.Recv()
	for _, index := range selection.Index() {
		if pointer, ok := owner.Underlying().(*types.Pointer); ok {
			owner = pointer.Elem()
		}
		if c.queries.Owns(owner) {
			return true
		}
		owner = owner.Underlying().(*types.Struct).Field(index).Type()
	}
	return false
}

// matchStatement finds the complete statement that owns a protected operand.
func (c *subsetChecks) matchStatement(expression ast.Expr) string {
	restricted := c.restrictedOperand(expression)
	for parent := c.parents[expression]; parent != nil; parent = c.parents[parent] {
		if typ, ok := parent.(ast.Expr); ok && c.pass.TypesInfo.Types[typ].IsType() {
			return "protected operation is outside the supported statement forms"
		}
		if declaration, ok := parent.(*ast.ValueSpec); ok {
			if restricted && (len(declaration.Names) != 1 || len(declaration.Values) > 1) &&
				!c.resultDeclaration(declaration) {
				return "multiple assignments involving protected storage are unsupported"
			}
			return c.matchDeclaration(declaration)
		}
		statement, ok := parent.(ast.Stmt)
		if !ok {
			continue
		}
		if restricted {
			if rule := c.matchRestrictedStatement(statement); rule != "" {
				return rule
			}
		}
		return c.matchStatementNode(statement)
	}
	return "protected operation is outside the supported statement forms"
}

// restrictedOperand includes Local transport, builders, and fixed Cell operations.
func (c *subsetChecks) restrictedOperand(expression ast.Expr) bool {
	typ := c.pass.TypesInfo.TypeOf(expression)
	if c.queries.Transport(typ) || c.queries.Callable(typ) || c.ownedExpression(expression) {
		return true
	}
	call, ok := ast.Unparen(expression).(*ast.CallExpr)
	return ok && c.ownedSlotCall(call)
}

// matchRestrictedStatement keeps the statement boundary for Local, builder, and Cell operations.
func (c *subsetChecks) matchRestrictedStatement(statement ast.Stmt) string {
	if clause, ok := c.parents[statement].(*ast.CommClause); ok && clause.Comm == statement {
		return "protected storage is unsupported in select communication clauses"
	}
	switch statement := statement.(type) {
	case *ast.AssignStmt:
		if (len(statement.Lhs) != 1 || len(statement.Rhs) != 1) && !c.resultAssignment(statement) {
			return "multiple assignments involving protected storage are unsupported"
		}
	case *ast.GoStmt, *ast.DeferStmt:
		return "protected storage requires synchronous calls"
	case *ast.SendStmt:
		return "protected operation is outside the supported statement forms"
	}
	return ""
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
		if switched, ok := c.parents[statement].(*ast.TypeSwitchStmt); ok && switched.Assign == statement {
			if assertion, ok := ast.Unparen(statement.X).(*ast.TypeAssertExpr); ok && assertion.Type == nil &&
				!c.restrictedOperand(assertion.X) {
				return ""
			}
		}
	case *ast.GoStmt:
		return c.matchCall(statement.Call)
	case *ast.DeferStmt:
		return c.matchCall(statement.Call)
	case *ast.SendStmt:
		if !c.protectedStorage(statement.Chan) {
			channel := c.pass.TypesInfo.TypeOf(statement.Chan).Underlying().(*types.Chan)
			return c.matchTransport(c.pass.TypesInfo.TypeOf(statement.Value), channel.Elem())
		}
	case *ast.CaseClause:
		if block, ok := c.parents[statement].(*ast.BlockStmt); ok {
			if parent, ok := c.parents[block].(*ast.SwitchStmt); ok {
				return c.matchSwitch(parent)
			}
		}
	}
	return "protected operation is outside the supported statement forms"
}
