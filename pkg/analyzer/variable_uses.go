package analyzer

import (
	"go/ast"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// variableUse records one package variable that a function uses directly or through a static callee.
type variableUse struct {
	Package string
	Name    string
	Var     string
}

// AFact identifies variableUse as a function-level analysis fact.
func (*variableUse) AFact() {}

// display qualifies the variable with its package name when it belongs to another package.
func (u *variableUse) display(pkg *types.Package) string {
	if u.Package == pkg.Path() {
		return u.Var
	}
	return u.Name + "." + u.Var
}

// variableUses finds, once per package, the functions that use a rejected package variable directly or through
// static calls, and exports one fact for each of them.
func (c *checks) variableUses() map[*types.Func]*variableUse {
	if c.uses != nil {
		return c.uses
	}
	c.uses = make(map[*types.Func]*variableUse)
	callers := make(map[*types.Func][]*types.Func)
	var found []*types.Func
	for _, file := range c.pass.Files {
		for _, declaration := range file.Decls {
			if function, ok := declaration.(*ast.FuncDecl); ok && function.Body != nil &&
				c.recordFunction(function, callers) {
				found = append(found, c.pass.TypesInfo.Defs[function.Name].(*types.Func))
			}
		}
	}
	for len(found) > 0 {
		callee := found[len(found)-1]
		found = found[:len(found)-1]
		for _, caller := range callers[callee] {
			if c.uses[caller] == nil {
				c.uses[caller] = c.uses[callee]
				found = append(found, caller)
			}
		}
	}
	if c.pass.ExportObjectFact != nil {
		for function, use := range c.uses {
			c.pass.ExportObjectFact(function, use)
		}
	}
	return c.uses
}

// recordFunction walks one function body and reports whether it uses a rejected variable or a marked callee.
func (c *checks) recordFunction(declaration *ast.FuncDecl, callers map[*types.Func][]*types.Func) bool {
	caller := c.pass.TypesInfo.Defs[declaration.Name].(*types.Func)
	ast.Inspect(declaration.Body, func(node ast.Node) bool {
		if id, ok := node.(*ast.Ident); ok && c.uses[caller] == nil {
			c.recordUse(caller, c.pass.TypesInfo.Uses[id], callers)
		}
		return true
	})
	return c.uses[caller] != nil
}

// recordUse records a rejected package variable, an imported callee fact, or a callee of this package.
func (c *checks) recordUse(caller *types.Func, object types.Object, callers map[*types.Func][]*types.Func) {
	if variable := packageVariable(object); variable != nil {
		if c.processVariableReason(variable) != "" {
			c.uses[caller] = &variableUse{Package: variable.Pkg().Path(), Name: variable.Pkg().Name(), Var: variable.Name()}
		}
		return
	}
	callee := staticFunction(object)
	if callee == nil || processMethod(callee) {
		return
	}
	if callee.Pkg() == c.pass.Pkg {
		callers[callee] = append(callers[callee], caller)
	} else if use := c.functionUse(nil, callee); use != nil {
		c.uses[caller] = use
	}
}

// functionUse returns the recorded use of a function of this package or the imported fact of another function.
func (c *checks) functionUse(uses map[*types.Func]*variableUse, function *types.Func) *variableUse {
	if function.Pkg() == c.pass.Pkg {
		return uses[function]
	}
	var use variableUse
	if c.pass.ImportObjectFact == nil || !c.pass.ImportObjectFact(function, &use) {
		return nil
	}
	return &use
}

// staticFunction returns the declared function or concrete method that a name refers to.
func staticFunction(object types.Object) *types.Func {
	function, ok := object.(*types.Func)
	if !ok || function.Pkg() == nil {
		return nil
	}
	function = function.Origin()
	if receiver := function.Type().(*types.Signature).Recv(); receiver != nil && types.IsInterface(receiver.Type()) {
		return nil
	}
	return function
}

// processMethod identifies a method of a process, which the process rules check where it is declared.
func processMethod(function *types.Func) bool {
	receiver := function.Type().(*types.Signature).Recv()
	if receiver == nil {
		return false
	}
	t := receiver.Type()
	if pointer, ok := t.Underlying().(*types.Pointer); ok {
		t = pointer.Elem()
	}
	return checker.Process(t)
}

// functionName names a function, or a method with its receiver type, and qualifies another package.
func (c *checks) functionName(function *types.Func) string {
	name := function.Name()
	if receiver := function.Type().(*types.Signature).Recv(); receiver != nil {
		t := receiver.Type()
		if pointer, ok := t.(*types.Pointer); ok {
			t = pointer.Elem()
		}
		if named, ok := types.Unalias(t).(*types.Named); ok {
			name = named.Obj().Name() + "." + name
		}
	}
	if function.Pkg() != c.pass.Pkg {
		name = function.Pkg().Name() + "." + name
	}
	return name
}

// calledName returns the name that a call expression calls.
func calledName(expression ast.Expr) *ast.Ident {
	switch e := ast.Unparen(expression).(type) {
	case *ast.Ident:
		return e
	case *ast.SelectorExpr:
		return e.Sel
	case *ast.IndexExpr:
		return calledName(e.X)
	case *ast.IndexListExpr:
		return calledName(e.X)
	}
	return nil
}
