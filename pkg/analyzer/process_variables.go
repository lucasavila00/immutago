package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

const processVariableFix = "; pass the value in the start arguments, or keep the state in a process " +
	"and publish it through an im.Owner field"

// checkProcessVariables checks each package variable that a process method uses.
func (c *checks) checkProcessVariables(declaration *ast.FuncDecl) {
	if declaration.Recv == nil || declaration.Body == nil {
		return
	}
	receiver := c.pass.TypesInfo.Defs[declaration.Name].Type().(*types.Signature).Recv().Type()
	if pointer, ok := receiver.Underlying().(*types.Pointer); ok {
		receiver = pointer.Elem()
	}
	if !checker.Process(receiver) {
		return
	}
	ast.Inspect(declaration.Body, func(node ast.Node) bool {
		if id, ok := node.(*ast.Ident); ok {
			if variable := packageVariable(c.pass.TypesInfo.Uses[id]); variable != nil {
				if reason := c.processVariableReason(variable); reason != "" {
					c.report(id.Pos(), "process method uses package variable "+c.variableName(variable)+reason+
						processVariableFix)
				}
			}
		}
		return true
	})
}

// packageVariable returns the object when it is a variable at package scope.
func packageVariable(object types.Object) *types.Var {
	variable, ok := object.(*types.Var)
	if !ok || variable.Pkg() == nil || variable.Pkg().Scope().Lookup(variable.Name()) != variable {
		return nil
	}
	return variable
}

// variableName qualifies a variable of another package with its package name.
func (c *checks) variableName(variable *types.Var) string {
	if variable.Pkg() == c.pass.Pkg {
		return variable.Name()
	}
	return variable.Pkg().Name() + "." + variable.Name()
}

// processVariableReason accepts an Owner slot and an unexported variable of this package whose type is an
// allowed stored type and that only package initialization changes.
func (c *checks) processVariableReason(variable *types.Var) string {
	switch {
	case checker.IsLibraryType(variable.Type(), "Owner"):
		return ""
	case variable.Pkg() != c.pass.Pkg:
		return " of another package, which any importer can change"
	case variable.Exported():
		return ", which other packages can change"
	case c.proofs.Reason(variable.Type()) != "":
		return ", whose type " + diagnosticType(variable.Type()) + " is not an allowed stored type"
	case c.changedVariables()[variable]:
		return ", which is changed after package initialization"
	}
	return ""
}

// changedVariables finds, once per package, the package variables that code outside init functions can change.
func (c *checks) changedVariables() map[*types.Var]bool {
	if c.changed != nil {
		return c.changed
	}
	c.changed = make(map[*types.Var]bool)
	for _, file := range c.pass.Files {
		for _, declaration := range file.Decls {
			if function, ok := declaration.(*ast.FuncDecl); ok && function.Recv == nil && function.Name.Name == "init" {
				continue
			}
			ast.Inspect(declaration, c.recordChange)
		}
	}
	return c.changed
}

// recordChange marks the variable under each Go operation that needs an addressable operand.
func (c *checks) recordChange(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.AssignStmt:
		for _, target := range n.Lhs {
			c.markChanged(target)
		}
	case *ast.IncDecStmt:
		c.markChanged(n.X)
	case *ast.RangeStmt:
		if n.Tok == token.ASSIGN {
			c.markChanged(n.Key)
			c.markChanged(n.Value)
		}
	case *ast.UnaryExpr:
		if n.Op == token.AND {
			c.markChanged(n.X)
		}
	case *ast.SliceExpr:
		if _, array := c.pass.TypesInfo.TypeOf(n.X).Underlying().(*types.Array); array {
			c.markChanged(n.X)
		}
	case *ast.SelectorExpr:
		if selection := c.pass.TypesInfo.Selections[n]; selection != nil && selection.Kind() == types.MethodVal &&
			addressedReceiver(selection) {
			c.markChanged(n.X)
		}
	}
	return true
}

// addressedReceiver identifies a pointer method selected on a value, which takes the address of that value.
func addressedReceiver(selection *types.Selection) bool {
	_, pointerMethod := selection.Obj().Type().(*types.Signature).Recv().Type().(*types.Pointer)
	_, pointerValue := selection.Recv().Underlying().(*types.Pointer)
	return pointerMethod && !pointerValue
}

// markChanged marks the package variable at the root of a selector, index, dereference, or parenthesis chain.
func (c *checks) markChanged(expression ast.Expr) {
	for {
		switch e := expression.(type) {
		case *ast.Ident:
			if variable := packageVariable(c.pass.TypesInfo.Uses[e]); variable != nil {
				c.changed[variable] = true
			}
			return
		case *ast.ParenExpr:
			expression = e.X
		case *ast.SelectorExpr:
			expression = e.X
		case *ast.IndexExpr:
			expression = e.X
		case *ast.StarExpr:
			expression = e.X
		default:
			return
		}
	}
}
