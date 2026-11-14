package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"
)

var scalarAtomics = map[string]bool{"Bool": true, "Int32": true, "Int64": true, "Uint32": true, "Uint64": true}

// atomicType identifies a named type of package sync/atomic.
func atomicType(t types.Type) bool {
	named, ok := types.Unalias(t).(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == "sync/atomic"
}

// atomicReason accepts an unexported scalar atomic of this package that the package uses only as the receiver
// of direct method calls.
func (c *checks) atomicReason(variable *types.Var) string {
	name := types.Unalias(variable.Type()).(*types.Named).Obj().Name()
	switch {
	case !scalarAtomics[name]:
		return ", whose type " + diagnosticType(variable.Type()) + " is not atomic.Bool, Int32, Int64, Uint32, or Uint64"
	case variable.Pkg() != c.pass.Pkg:
		return " of another package, which any importer can change"
	case variable.Exported():
		return ", which other packages can change"
	case c.atomicCounters()[variable]:
		return ""
	}
	return ", which the package uses other than as the receiver of a direct method call"
}

// atomicCounters finds, once per package, the scalar atomic package variables whose every use is the receiver
// of a direct method call.
func (c *checks) atomicCounters() map[*types.Var]bool {
	if c.counters != nil {
		return c.counters
	}
	receivers := c.methodReceivers()
	c.counters = make(map[*types.Var]bool)
	for id, object := range c.pass.TypesInfo.Uses {
		variable := packageVariable(object)
		if variable == nil || variable.Pkg() != c.pass.Pkg || !atomicType(variable.Type()) {
			continue
		}
		if valid, seen := c.counters[variable]; !seen || valid {
			c.counters[variable] = receivers[id]
		}
	}
	return c.counters
}

// methodReceivers finds the names that are the receiver of a direct method call.
func (c *checks) methodReceivers() map[*ast.Ident]bool {
	receivers := make(map[*ast.Ident]bool)
	for _, file := range c.pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				if selector, ok := call.Fun.(*ast.SelectorExpr); ok && c.pass.TypesInfo.Selections[selector] != nil {
					if id, ok := selector.X.(*ast.Ident); ok {
						receivers[id] = true
					}
				}
			}
			return true
		})
	}
	return receivers
}

// changedError marks an exported error variable that its own package changes after package initialization.
type changedError struct{}

// AFact identifies changedError as a variable-level analysis fact.
func (*changedError) AFact() {}

// errorVariable identifies a variable of the predeclared error type.
func errorVariable(variable *types.Var) bool {
	return types.Identical(variable.Type(), types.Universe.Lookup("error").Type())
}

// errorReason accepts an error variable of any package as a compared operand when no checked code changes it.
func (c *checks) errorReason(id *ast.Ident, variable *types.Var) string {
	switch {
	case !c.comparedNames()[id]:
		return ", which a process method can only compare with ==, !=, a switch case, or errors.Is"
	case variable.Pkg() == c.pass.Pkg && c.changedVariables()[variable]:
		return ", which is changed after package initialization"
	case variable.Pkg() != c.pass.Pkg && c.pass.ImportObjectFact != nil &&
		c.pass.ImportObjectFact(variable, new(changedError)):
		return ", which its package changes after package initialization"
	}
	return ""
}

// comparedNames finds, once per package, the names that are an operand of == or !=, a value of a switch case
// with a tag, or the second argument of errors.Is.
func (c *checks) comparedNames() map[*ast.Ident]bool {
	if c.compared != nil {
		return c.compared
	}
	c.compared = make(map[*ast.Ident]bool)
	for _, file := range c.pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.BinaryExpr:
				if n.Op == token.EQL || n.Op == token.NEQ {
					c.markCompared(n.X)
					c.markCompared(n.Y)
				}
			case *ast.SwitchStmt:
				c.markCases(n)
			case *ast.CallExpr:
				if function, ok := c.pass.TypesInfo.Uses[calledName(n.Fun)].(*types.Func); ok && len(n.Args) == 2 &&
					function.Pkg() != nil && function.Pkg().Path() == "errors" && function.Name() == "Is" {
					c.markCompared(n.Args[1])
				}
			}
			return true
		})
	}
	return c.compared
}

// markCases marks the values of the case clauses of a switch with a tag.
func (c *checks) markCases(statement *ast.SwitchStmt) {
	if statement.Tag == nil {
		return
	}
	for _, clause := range statement.Body.List {
		for _, value := range clause.(*ast.CaseClause).List {
			c.markCompared(value)
		}
	}
}

// markCompared marks a variable name or a qualified variable name.
func (c *checks) markCompared(expression ast.Expr) {
	switch e := ast.Unparen(expression).(type) {
	case *ast.Ident:
		c.compared[e] = true
	case *ast.SelectorExpr:
		if c.pass.TypesInfo.Selections[e] == nil {
			c.compared[e.Sel] = true
		}
	}
}

// exportChangedErrors exports a fact for each exported error variable that this package changes after package
// initialization, so that importers do not accept it as a compared operand.
func (c *checks) exportChangedErrors() {
	if c.pass.ExportObjectFact == nil {
		return
	}
	scope := c.pass.Pkg.Scope()
	for _, name := range scope.Names() {
		variable, ok := scope.Lookup(name).(*types.Var)
		if ok && variable.Exported() && errorVariable(variable) && c.changedVariables()[variable] {
			c.pass.ExportObjectFact(variable, new(changedError))
		}
	}
}

// checkForeignErrorChange rejects an assignment to, or the address of, an error variable of another package.
func (c *checks) checkForeignErrorChange(node ast.Node) {
	var targets []ast.Expr
	switch n := node.(type) {
	case *ast.AssignStmt:
		targets = n.Lhs
	case *ast.RangeStmt:
		if n.Tok == token.ASSIGN {
			targets = []ast.Expr{n.Key, n.Value}
		}
	case *ast.UnaryExpr:
		if n.Op == token.AND {
			targets = []ast.Expr{n.X}
		}
	}
	for _, target := range targets {
		var id *ast.Ident
		switch e := ast.Unparen(target).(type) {
		case *ast.Ident:
			id = e
		case *ast.SelectorExpr:
			id = e.Sel
		}
		if id == nil {
			continue
		}
		if variable := packageVariable(c.pass.TypesInfo.Uses[id]); variable != nil && variable.Pkg() != c.pass.Pkg &&
			errorVariable(variable) {
			c.report(id.Pos(), "checked code cannot assign or take the address of error variable "+c.variableName(variable)+
				" of another package; declare a variable in this package")
		}
	}
}
