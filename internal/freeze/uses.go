package freeze

import (
	"go/ast"
	"go/token"
	"go/types"
)

// checkUses rejects every handle-aliasing operation, including aliases created before freezing.
func (p *proof) checkUses() {
	for _, file := range p.pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			id, ok := node.(*ast.Ident)
			if !ok {
				return true
			}
			variable, _ := p.pass.TypesInfo.Uses[id].(*types.Var)
			item, tracked := p.owners[variable]
			if !tracked {
				return true
			}
			if p.body(id) != item.body {
				p.reject(id, variable, "capturing or using a builder outside its declaring function")
				return true
			}
			expression, parent := p.unparenParent(id)
			if !p.allowedUse(expression, parent, variable) {
				p.reject(id, variable, "copying, rebinding, slicing, addressing, or escaping its native handle")
			}
			return true
		})
	}
}

// reject explains the conservative unique-owner boundary rather than silently assuming an alias is dead.
func (p *proof) reject(node ast.Node, variable *types.Var, reason string) {
	p.report(node.Pos(), "unsupported owned container "+variable.Name()+": "+reason+
		"; keep one local handle, edit its elements, then Freeze it; use Clone for an independent builder")
}

// unparenParent skips syntax-only parentheses when classifying a handle use.
func (p *proof) unparenParent(expression ast.Expr) (ast.Expr, ast.Node) {
	for {
		parent := p.parents[expression]
		if paren, ok := parent.(*ast.ParenExpr); ok {
			expression = paren
			continue
		}
		return expression, parent
	}
}

// allowedUse permits operations that cannot retain another native handle.
func (p *proof) allowedUse(expression ast.Expr, parent ast.Node, variable *types.Var) bool {
	switch node := parent.(type) {
	case *ast.IndexExpr:
		return node.X == expression && p.safeElement(node)
	case *ast.RangeStmt:
		return node.X == expression
	case *ast.CallExpr:
		return p.allowedCall(node, expression, variable)
	case *ast.AssignStmt:
		return len(node.Lhs) == 1 && node.Lhs[0] == expression && p.appendAssignment(node, variable)
	}
	return false
}

// safeElement rejects explicit and implicit addresses into storage before they can outlive the builder.
func (p *proof) safeElement(node ast.Node) bool {
	for {
		parent := p.parents[node]
		switch next := parent.(type) {
		case *ast.ParenExpr:
		case *ast.SelectorExpr:
			if next.X != node {
				return true
			}
			if p.pointerMethod(next) {
				return false
			}
		case *ast.IndexExpr:
			if next.X != node {
				return true
			}
		case *ast.StarExpr:
		case *ast.UnaryExpr:
			return next.Op != token.AND
		case *ast.SliceExpr:
			return next.X != node
		case *ast.RangeStmt:
			return next.X != node
		default:
			return true
		}
		node = parent
	}
}

// allowedCall admits built-in element operations and synchronous ownership consumption.
func (p *proof) allowedCall(call *ast.CallExpr, expression ast.Expr, variable *types.Var) bool {
	if _, deferred := p.parents[call].(*ast.DeferStmt); deferred {
		return false
	}
	if _, concurrent := p.parents[call].(*ast.GoStmt); concurrent {
		return false
	}
	if freezing(p.pass, call) {
		if len(call.Args) != 1 || call.Args[0] != expression {
			return false
		}
		return !p.activeRange(call, variable)
	}
	builtin, ok := calledObject(p.pass, call.Fun).(*types.Builtin)
	if !ok {
		return false
	}
	switch builtin.Name() {
	case "len", "cap", "clear":
		return len(call.Args) == 1 && call.Args[0] == expression
	case "delete":
		return len(call.Args) == 2 && call.Args[0] == expression
	case "copy":
		return len(call.Args) == 2
	case "append":
		if call.Ellipsis.IsValid() || len(call.Args) == 0 || call.Args[0] != expression {
			return false
		}
		outer, parent := p.unparenParent(call)
		assignment, ok := parent.(*ast.AssignStmt)
		return ok && len(assignment.Rhs) == 1 && assignment.Rhs[0] == outer && p.appendAssignment(assignment, variable)
	}
	return false
}

// appendAssignment permits only updating the same unique slice handle with append's result.
func (p *proof) appendAssignment(statement *ast.AssignStmt, variable *types.Var) bool {
	if statement.Tok != token.ASSIGN || len(statement.Lhs) != 1 || len(statement.Rhs) != 1 {
		return false
	}
	id, ok := ast.Unparen(statement.Lhs[0]).(*ast.Ident)
	if !ok || p.pass.TypesInfo.ObjectOf(id) != variable {
		return false
	}
	call, ok := ast.Unparen(statement.Rhs[0]).(*ast.CallExpr)
	if !ok || call.Ellipsis.IsValid() || len(call.Args) == 0 {
		return false
	}
	builtin, ok := calledObject(p.pass, call.Fun).(*types.Builtin)
	if !ok || builtin.Name() != "append" {
		return false
	}
	source, ok := ast.Unparen(call.Args[0]).(*ast.Ident)
	return ok && p.pass.TypesInfo.ObjectOf(source) == variable
}

// activeRange rejects consumption while a native range iterator still retains the handle.
func (p *proof) activeRange(node ast.Node, variable *types.Var) bool {
	for node = p.parents[node]; node != nil; node = p.parents[node] {
		loop, ok := node.(*ast.RangeStmt)
		if !ok {
			continue
		}
		id, ok := ast.Unparen(loop.X).(*ast.Ident)
		if ok && p.pass.TypesInfo.ObjectOf(id) == variable {
			return true
		}
	}
	return false
}

// pointerMethod identifies implicit addresses taken by pointer-receiver method selections.
func (p *proof) pointerMethod(node ast.Node) bool {
	selector, ok := node.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	selection := p.pass.TypesInfo.Selections[selector]
	if selection == nil {
		return false
	}
	signature, _ := selection.Obj().Type().(*types.Signature)
	if signature == nil || signature.Recv() == nil {
		return false
	}
	_, pointer := signature.Recv().Type().Underlying().(*types.Pointer)
	return pointer
}
