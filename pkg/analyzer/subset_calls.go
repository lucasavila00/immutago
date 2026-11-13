package analyzer

import (
	"go/ast"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// calledObject resolves a direct function or method through generic syntax.
func (c *subsetChecks) calledObject(e ast.Expr) types.Object {
	switch x := ast.Unparen(e).(type) {
	case *ast.Ident:
		return c.pass.TypesInfo.ObjectOf(x)
	case *ast.SelectorExpr:
		return c.pass.TypesInfo.ObjectOf(x.Sel)
	case *ast.IndexExpr:
		return c.calledObject(x.X)
	case *ast.IndexListExpr:
		return c.calledObject(x.X)
	}
	return nil
}

// constructorInputs permits only copied inline literals at the sealed constructor boundary.
func (c *subsetChecks) constructorInputs() {
	for node := range c.parents {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			continue
		}
		obj := c.calledObject(call.Fun)
		if !libraryObject(obj) || (obj.Name() != "NewMap" && obj.Name() != "NewSlice") {
			continue
		}
		for _, arg := range call.Args {
			literal, ok := ast.Unparen(arg).(*ast.CompositeLit)
			if !ok {
				continue
			}
			switch c.pass.TypesInfo.TypeOf(literal).Underlying().(type) {
			case *types.Map, *types.Slice:
				c.exempt[literal] = true
				ast.Inspect(literal.Type, func(n ast.Node) bool {
					if n != nil {
						c.exempt[n] = true
					}
					return true
				})
			}
		}
	}
}

// call restricts shared arguments to direct synchronous calls with exact typed parameters.
func (c *subsetChecks) call(call *ast.CallExpr) {
	obj := c.calledObject(call.Fun)
	if c.pass.TypesInfo.Types[call.Fun].IsType() {
		if protectedReference(c.pass.TypesInfo.TypeOf(call)) {
			c.report(call, "shared pointer conversions are unsupported")
		}
		if len(call.Args) == 1 {
			c.transport(call.Args[0], c.pass.TypesInfo.TypeOf(call))
		}
		return
	}
	if builtin, ok := obj.(*types.Builtin); ok && builtin.Name() == "new" &&
		protectedReference(c.pass.TypesInfo.TypeOf(call)) {
		c.report(call, "pointer-owned construction is unsupported")
		return
	}
	signature, _ := c.pass.TypesInfo.TypeOf(call.Fun).Underlying().(*types.Signature)
	if signature == nil {
		return
	}
	if len(call.Args) == 1 {
		var destinations []types.Type
		for i := 0; i < signature.Params().Len(); i++ {
			destinations = append(destinations, signature.Params().At(i).Type())
		}
		c.tupleTransport(call.Args[0], destinations)
	}
	direct, static := obj.(*types.Func)
	if sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr); ok {
		if selection := c.pass.TypesInfo.Selections[sel]; selection != nil {
			method, _ := selection.Obj().Type().(*types.Signature)
			if method != nil && method.Recv() != nil {
				if _, pointer := method.Recv().Type().Underlying().(*types.Pointer); pointer && c.protectedRoot(sel.X) {
					c.synchronous(call)
					if genericMethod(selection.Obj()) && !libraryObject(obj) {
						c.report(call, "generic borrowing calls are unsupported")
					}
					if !checker.Contains(c.pass.TypesInfo.TypeOf(sel.X)) {
						c.report(call, "scalar interior pointer methods are unsupported")
					}
				}
			}

			if _, iface := selection.Recv().Underlying().(*types.Interface); iface {
				static = false
			}
			if protectedReference(c.pass.TypesInfo.TypeOf(sel.X)) {
				c.synchronous(call)
			}
		}
	}
	for i, arg := range call.Args {
		t := c.pass.TypesInfo.TypeOf(arg)
		if !checker.Contains(t) && !protectedCallable(t) {
			continue
		}
		if !static && !libraryObject(obj) {
			c.report(arg, "dynamic calls with immutable storage are unsupported")
		}
		if protectedReference(t) {
			c.synchronous(call)
			if libraryObject(obj) || (direct != nil && genericSignature(direct.Origin().Type().(*types.Signature))) {
				c.report(arg, "generic borrowing calls are unsupported")
			}
		}
		parameter := i
		if signature.Variadic() && parameter >= signature.Params().Len()-1 {
			parameter = signature.Params().Len() - 1
		}
		if parameter >= 0 && parameter < signature.Params().Len() {
			c.transport(arg, signature.Params().At(parameter).Type())
		}
	}
	if libraryObject(obj) && obj.Name() == "All" && protectedCallable(c.pass.TypesInfo.TypeOf(call)) {
		r, ok := c.parents[call].(*ast.RangeStmt)
		if !ok || r.X != call {
			c.report(call, "immutable iterators must be consumed by direct range")
		}
	}
}

// synchronous rejects borrows retained by deferred or concurrent execution.
func (c *subsetChecks) synchronous(call *ast.CallExpr) {
	switch c.parents[call].(type) {
	case *ast.DeferStmt, *ast.GoStmt:
		c.report(call, "shared references require synchronous calls")
	}
}

// transport rejects interface boxing and callback or container transport of immutable storage.
func (c *subsetChecks) transport(value ast.Expr, destination types.Type) {
	c.transportType(value, c.pass.TypesInfo.TypeOf(value), destination)
}

// transportType checks one result of a value or tuple expression.
func (c *subsetChecks) transportType(value ast.Expr, source, destination types.Type) {
	if !checker.Contains(source) && !protectedCallable(source) {
		return
	}
	if destination == nil {
		return
	}
	switch destination.Underlying().(type) {
	case *types.Interface:
		c.report(value, "boxing immutable storage in interfaces is unsupported")
	case *types.Signature:
		if protectedCallable(source) {
			c.report(value, "immutable callbacks are unsupported")
		}
	case *types.Slice, *types.Map, *types.Chan:
		if protectedReference(source) {
			c.report(value, "shared references cannot enter containers")
		}
	}
}

// protectedCallable detects callbacks whose signature can carry immutable storage.
func protectedCallable(t types.Type) bool { return callableContains(t, make(map[types.Type]bool)) }

// callableContains walks recursive callback signatures once.
func callableContains(t types.Type, seen map[types.Type]bool) bool {
	if t == nil || seen[t] {
		return false
	}
	seen[t] = true
	sig, ok := t.Underlying().(*types.Signature)
	if !ok {
		return false
	}
	for _, tuple := range []*types.Tuple{sig.Params(), sig.Results()} {
		for i := 0; i < tuple.Len(); i++ {
			item := tuple.At(i).Type()
			if checker.Contains(item) || callableContains(item, seen) {
				return true
			}
		}
	}
	return false
}

// methodValue rejects bound method references that retain protected receivers.
func (c *subsetChecks) methodValue(sel *ast.SelectorExpr) {
	selection := c.pass.TypesInfo.Selections[sel]
	if selection == nil || selection.Kind() != types.MethodVal || !c.protectedRoot(sel.X) {
		return
	}
	if call, ok := c.parents[sel].(*ast.CallExpr); ok && call.Fun == sel {
		return
	}
	c.report(sel, "bound immutable methods are unsupported")
}

// closure rejects implicit reference captures of protected local storage.
func (c *subsetChecks) closure(literal *ast.FuncLit) {
	ast.Inspect(literal.Body, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		obj, ok := c.pass.TypesInfo.Uses[id].(*types.Var)
		if ok && obj.Pos() < literal.Pos() && checker.Contains(obj.Type()) {
			c.report(id, "closures cannot capture immutable storage")
		}
		return true
	})
}

// globalStore prevents a shared local reference from entering package storage.
func (c *subsetChecks) globalStore(destination, value ast.Expr) {
	if !protectedReference(c.pass.TypesInfo.TypeOf(value)) {
		return
	}
	id, ok := ast.Unparen(destination).(*ast.Ident)
	if !ok {
		c.report(destination, "shared references cannot enter aggregate storage")
		return
	}
	obj := c.pass.TypesInfo.ObjectOf(id)
	if obj != nil && obj.Parent() == c.pass.Pkg.Scope() {
		c.report(destination, "global shared references are unsupported")
	}
}

// returnType finds the declared result type for implicit boxing checks.
func (c *subsetChecks) returnType(statement *ast.ReturnStmt, index int) types.Type {
	for parent := c.parents[statement]; parent != nil; parent = c.parents[parent] {
		var t types.Type
		switch fn := parent.(type) {
		case *ast.FuncDecl:
			t = c.pass.TypesInfo.Defs[fn.Name].Type()
		case *ast.FuncLit:
			t = c.pass.TypesInfo.TypeOf(fn)
		}
		if t == nil {
			continue
		}
		signature := t.Underlying().(*types.Signature)
		if index < signature.Results().Len() {
			return signature.Results().At(index).Type()
		}
		return nil
	}
	return nil
}

// tupleTransport checks implicit boxing when one expression supplies several results.
func (c *subsetChecks) tupleTransport(value ast.Expr, destinations []types.Type) {
	tuple, ok := c.pass.TypesInfo.TypeOf(value).(*types.Tuple)
	if !ok {
		return
	}
	for i := 0; i < tuple.Len() && i < len(destinations); i++ {
		c.transportType(value, tuple.At(i).Type(), destinations[i])
	}
}

// genericSignature includes receiver parameters, which method instantiation retains separately.
func genericSignature(signature *types.Signature) bool {
	return signature.TypeParams().Len() > 0 || signature.RecvTypeParams().Len() > 0
}

// genericMethod uses the origin because an instantiated selection can erase receiver parameters.
func genericMethod(obj types.Object) bool {
	function, ok := obj.(*types.Func)
	return ok && genericSignature(function.Origin().Type().(*types.Signature))
}
