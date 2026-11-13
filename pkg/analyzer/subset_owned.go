package analyzer

import (
	"go/ast"
	"go/types"

	"github.com/lucasavila00/immutago/internal/freeze"
)

// ownedExpression admits native handles only when the ownership checker checks their uses.
func (c *subsetChecks) ownedExpression(expression ast.Expr) bool {
	if id, ok := ast.Unparen(expression).(*ast.Ident); ok {
		variable, _ := c.pass.TypesInfo.ObjectOf(id).(*types.Var)
		return c.owners[variable]
	}
	return false
}

// ownedInputs admits container syntax at checked local initialization and freeze boundaries.
func (c *subsetChecks) ownedInputs() {
	for node := range c.parents {
		switch node := node.(type) {
		case *ast.AssignStmt:
			for i, value := range node.Rhs {
				if i < len(node.Lhs) && c.ownedExpression(node.Lhs[i]) {
					c.ownedInput(value)
				}
			}
		case *ast.ValueSpec:
			c.ownedDeclaration(node)
		case *ast.CallExpr:
			object := c.calledObject(node.Fun)
			if libraryObject(object) && (object.Name() == "FreezeMap" || object.Name() == "FreezeSlice") {
				for _, value := range node.Args {
					c.ownedInput(value)
				}
			}
		}
	}
}

// ownedDeclaration admits the native type syntax and initializer of checked local builders.
func (c *subsetChecks) ownedDeclaration(node *ast.ValueSpec) {
	for i, name := range node.Names {
		if !c.ownedExpression(name) {
			continue
		}
		c.exemptType(node.Type)
		if i < len(node.Values) {
			c.ownedInput(node.Values[i])
		}
	}
}

// ownedInput exempts the native handle expression without exempting its element operations.
func (c *subsetChecks) ownedInput(expression ast.Expr) {
	c.exempt[expression] = true
	switch expression := expression.(type) {
	case *ast.ParenExpr:
		c.ownedInput(expression.X)
	case *ast.CompositeLit:
		c.exemptType(expression.Type)
	case *ast.CallExpr:
		if !freeze.Fresh(c.pass, expression) {
			return
		}
		if id, ok := ast.Unparen(expression.Fun).(*ast.Ident); ok {
			if object, ok := c.pass.TypesInfo.ObjectOf(id).(*types.Builtin); ok && object.Name() == "make" {
				c.exemptType(expression.Args[0])
			}
		}
	}
}

// exemptType suppresses only the native-container boundary within a checked type expression.
func (c *subsetChecks) exemptType(expression ast.Expr) {
	if expression == nil {
		return
	}
	ast.Inspect(expression, func(n ast.Node) bool {
		if n != nil {
			c.exempt[n] = true
		}
		return true
	})
}

// freezeFunctionUse prevents indirect calls from hiding the ownership transition.
func (c *subsetChecks) freezeFunctionUse(expression ast.Expr) {
	if parent, ok := c.parents[expression].(*ast.SelectorExpr); ok && parent.Sel == expression {
		return
	}
	object := c.calledObject(expression)
	if !libraryObject(object) || (object.Name() != "FreezeMap" && object.Name() != "FreezeSlice") {
		return
	}
	call := c.directFreezeCall(expression)
	if call == nil {
		c.report(expression, "freeze functions must be called directly")
		return
	}
	switch c.parents[call].(type) {
	case *ast.GoStmt, *ast.DeferStmt:
		c.report(call, "freezing requires a synchronous call")
	}
}

// directFreezeCall follows only parentheses and generic instantiation into a direct callee.
func (c *subsetChecks) directFreezeCall(node ast.Node) *ast.CallExpr {
	for {
		switch parent := c.parents[node].(type) {
		case *ast.ParenExpr:
			node = parent
		case *ast.IndexExpr:
			if parent.X != node {
				return nil
			}
			node = parent
		case *ast.IndexListExpr:
			if parent.X != node {
				return nil
			}
			node = parent
		case *ast.CallExpr:
			if parent.Fun == node {
				return parent
			}
			return nil
		default:
			return nil
		}
	}
}

// ownedBuiltin admits non-escaping native operations whose handle uses the ownership checker verifies.
func (c *subsetChecks) ownedBuiltin(call *ast.CallExpr) bool {
	builtin, ok := c.calledObject(call.Fun).(*types.Builtin)
	if !ok || len(call.Args) == 0 {
		return false
	}
	switch builtin.Name() {
	case "len", "cap", "append", "clear", "delete", "copy":
		return c.ownedExpression(call.Args[0])
	}
	return false
}
