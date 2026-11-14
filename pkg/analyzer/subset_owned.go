package analyzer

import (
	"go/ast"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
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

// standaloneFreeze admits only a single named frozen result or an explicit standalone discard.
func (c *subsetChecks) standaloneFreeze(call *ast.CallExpr) bool {
	var expression ast.Expr = call
	for {
		parent, ok := c.parents[expression].(*ast.ParenExpr)
		if !ok {
			break
		}
		expression = parent
	}
	switch statement := c.parents[expression].(type) {
	case *ast.AssignStmt:
		if len(statement.Lhs) != 1 || len(statement.Rhs) != 1 || statement.Rhs[0] != expression {
			return false
		}
		name, ok := statement.Lhs[0].(*ast.Ident)
		return ok && (name.Name == "_" || checker.IsCollection(c.pass.TypesInfo.TypeOf(name)))
	case *ast.ValueSpec:
		return len(statement.Names) == 1 && len(statement.Values) == 1 && statement.Values[0] == expression
	}
	return false
}
