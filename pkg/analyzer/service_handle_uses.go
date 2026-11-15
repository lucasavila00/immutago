package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"
)

// matchHandleUse lists consumers that preserve the concrete handle and its accessor checks.
func (c *subsetChecks) matchHandleUse(expression ast.Expr) string {
	if call, ok := expression.(*ast.CallExpr); ok && c.pass.TypesInfo.Types[call.Fun].IsType() {
		return c.handleCall(call)
	}
	if _, assertion := expression.(*ast.TypeAssertExpr); assertion {
		return "service handles cannot be recovered from interfaces; use a concrete Ref type"
	}
	if statement, ok := c.parents[expression].(ast.Stmt); ok {
		return c.handleStatement(statement)
	}
	switch parent := c.parents[expression].(type) {
	case *ast.ParenExpr, *ast.SelectorExpr, *ast.StarExpr, *ast.SliceExpr, *ast.IndexListExpr:
		return ""
	case *ast.UnaryExpr:
		if parent.Op == token.AND || parent.Op == token.ARROW {
			return ""
		}
	case *ast.IndexExpr:
		if ast.Unparen(parent.X) == ast.Unparen(expression) {
			return ""
		}
		if storage, ok := c.pass.TypesInfo.TypeOf(parent.X).Underlying().(*types.Map); ok {
			return c.matchTransport(c.pass.TypesInfo.TypeOf(expression), storage.Key())
		}
	case *ast.ValueSpec:
		destinations := make([]ast.Expr, len(parent.Names))
		for i, name := range parent.Names {
			destinations[i] = name
		}
		return c.handleBindings(parent.Values, destinations)
	case *ast.CallExpr:
		return c.handleCall(parent)
	case *ast.CompositeLit:
		return c.matchLiteral(parent)
	case *ast.KeyValueExpr:
		if literal, ok := c.parents[parent].(*ast.CompositeLit); ok {
			return c.matchLiteral(literal)
		}
	case *ast.BinaryExpr:
		return c.matchBinary(parent)
	}
	return "service handle operation is outside the allowed concrete copy, storage, and call forms"
}

// handleStatement checks concrete copies and each implicit statement conversion.
func (c *subsetChecks) handleStatement(statement ast.Stmt) string {
	switch parent := statement.(type) {
	case *ast.ExprStmt, *ast.GoStmt, *ast.DeferStmt:
		return ""
	case *ast.AssignStmt:
		return c.handleBindings(parent.Rhs, parent.Lhs)
	case *ast.ReturnStmt:
		return c.matchReturn(parent)
	case *ast.SwitchStmt:
		return c.matchSwitch(parent)
	case *ast.CaseClause:
		if block, ok := c.parents[parent].(*ast.BlockStmt); ok {
			if statement, ok := c.parents[block].(*ast.SwitchStmt); ok {
				return c.matchSwitch(statement)
			}
		}
	case *ast.SendStmt:
		channel := c.pass.TypesInfo.TypeOf(parent.Chan).Underlying().(*types.Chan)
		return c.matchTransport(c.pass.TypesInfo.TypeOf(parent.Value), channel.Elem())
	case *ast.RangeStmt:
		return c.handleRange(parent)
	}
	return "service handle operation is outside the allowed concrete copy, storage, and call forms"
}

// handleBindings checks implicit conversions in single, multiple, and tuple assignments.
func (c *subsetChecks) handleBindings(sources, destinations []ast.Expr) string {
	for i, destination := range destinations {
		if len(sources) == 0 {
			return ""
		}
		var source types.Type
		if len(sources) == len(destinations) {
			source = c.pass.TypesInfo.TypeOf(sources[i])
		} else if tuple, ok := c.pass.TypesInfo.TypeOf(sources[0]).(*types.Tuple); ok {
			source = tuple.At(i).Type()
		} else if i == 0 {
			source = c.pass.TypesInfo.TypeOf(sources[0])
		}
		if rule := c.matchTransport(source, c.pass.TypesInfo.TypeOf(destination)); rule != "" {
			return rule
		}
	}
	return ""
}

// handleCall checks concrete arguments and disallows conversions that erase handle identity.
func (c *subsetChecks) handleCall(call *ast.CallExpr) string {
	if c.pass.TypesInfo.Types[call.Fun].IsType() {
		return "service handle conversions are unsupported; use a type alias or copy the Ref value"
	}
	if signature, ok := c.pass.TypesInfo.TypeOf(call.Fun).Underlying().(*types.Signature); ok {
		return c.matchCallArguments(call, signature)
	}
	return "service handles require a typed call"
}

// handleRange checks conversions from native or iterator values into existing range variables.
func (c *subsetChecks) handleRange(statement *ast.RangeStmt) string {
	typ := c.pass.TypesInfo.TypeOf(statement.X)
	if pointer, ok := typ.Underlying().(*types.Pointer); ok {
		typ = pointer.Elem()
	}
	yielded := rangeTypes(typ)
	if channel, ok := typ.Underlying().(*types.Chan); ok {
		yielded[0] = channel.Elem()
	}
	for i, target := range []ast.Expr{statement.Key, statement.Value} {
		if target != nil {
			if rule := c.matchTransport(yielded[i], c.pass.TypesInfo.TypeOf(target)); rule != "" {
				return rule
			}
		}
	}
	return ""
}
