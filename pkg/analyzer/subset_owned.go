package analyzer

import (
	"go/ast"
	"go/token"
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

// matchBuilderBinding accepts fresh local construction or append into the same existing handle.
func (c *subsetChecks) matchBuilderBinding(destination, source ast.Expr, declaration bool) checker.Failure {
	if !c.ownedExpression(destination) {
		return checker.Failure{}
	}
	if declaration && freeze.Fresh(c.pass, source) {
		return checker.Failure{}
	}
	if !declaration {
		if call, ok := ast.Unparen(source).(*ast.CallExpr); ok {
			if builtin, ok := c.calledObject(call.Fun).(*types.Builtin); ok && builtin.Name() == "append" &&
				c.matchBuilderBuiltin(call, "append") {
				return checker.Failure{}
			}
		}
	}
	return shapeFailure("native builders require fresh local bindings or append to the same handle")
}

// checkBuilderTypes proves protected keys and elements once for each tracked owner.
func (c *subsetChecks) checkBuilderTypes() {
	for variable := range c.owners {
		if failure := c.builderTypeFailure(variable.Type()); failure.Reason != "" {
			c.pass.Reportf(variable.Pos(), "unsupported native builder: %s\n  fix: %s", failure.Reason, failure.Repair)
		}
	}
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

// builderTypeFailure proves keys and elements before the owned native handle is exempted.
func (c *subsetChecks) builderTypeFailure(t types.Type) checker.Failure {
	if !c.queries.Contains(t) {
		return checker.Failure{}
	}
	switch storage := t.Underlying().(type) {
	case *types.Slice:
		return c.proofs.Failure(storage.Elem())
	case *types.Map:
		if failure := c.proofs.Failure(storage.Key()); failure.Reason != "" {
			return failure
		}
		return c.proofs.Failure(storage.Elem())
	}
	return checker.Failure{}
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

// matchBuilderBuiltin matches the complete native builtin form before the ownership proof applies.
func (c *subsetChecks) matchBuilderBuiltin(call *ast.CallExpr, name string) bool {
	if len(call.Args) == 0 {
		return false
	}
	switch name {
	case "len", "cap", "clear":
		return len(call.Args) == 1 && c.ownedExpression(call.Args[0])
	case "delete":
		return len(call.Args) == 2 && c.ownedExpression(call.Args[0])
	case "copy":
		return len(call.Args) == 2 && (c.ownedExpression(call.Args[0]) || c.ownedExpression(call.Args[1]))
	case "append":
		if call.Ellipsis.IsValid() || !c.ownedExpression(call.Args[0]) {
			return false
		}
		statement, ok := c.unparenthesizedParent(call).(*ast.AssignStmt)
		if !ok || statement.Tok != token.ASSIGN || len(statement.Lhs) != 1 || len(statement.Rhs) != 1 {
			return false
		}
		target, ok := ast.Unparen(statement.Lhs[0]).(*ast.Ident)
		source, sourceOK := ast.Unparen(call.Args[0]).(*ast.Ident)
		return ok && sourceOK && ast.Unparen(statement.Rhs[0]) == call &&
			c.pass.TypesInfo.ObjectOf(target) == c.pass.TypesInfo.ObjectOf(source)
	}
	return false
}

// standaloneFreeze requires one destination; assignment checks enforce types and write permissions.
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
		switch ast.Unparen(statement.Lhs[0]).(type) {
		case *ast.Ident, *ast.SelectorExpr, *ast.IndexExpr, *ast.StarExpr:
			return true
		default:
			return false
		}
	case *ast.ValueSpec:
		return len(statement.Names) == 1 && len(statement.Values) == 1 && statement.Values[0] == expression
	}
	return false
}
