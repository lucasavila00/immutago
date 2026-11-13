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

// call checks direct calls, conversions, and the types of transported arguments.
func (c *subsetChecks) call(call *ast.CallExpr) {
	obj := c.calledObject(call.Fun)
	if builtin, ok := obj.(*types.Builtin); ok && (builtin.Name() == "make" || builtin.Name() == "new") {
		return
	}
	if c.ownedBuiltin(call) {
		return
	}
	if c.pass.TypesInfo.Types[call.Fun].IsType() {
		if protectedReference(c.pass.TypesInfo.TypeOf(call)) {
			c.report(call, "shared pointer conversions are unsupported")
		}
		if len(call.Args) == 1 {
			c.transport(call.Args[0], c.pass.TypesInfo.TypeOf(call))
		}
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
	_, static := obj.(*types.Func)
	static = c.methodCall(call, static)
	if !static && protectedCallable(signature) {
		c.report(call, "dynamic calls with immutable storage are unsupported")
	}
	c.callArguments(call, obj, signature, static)

	c.iteratorResult(call, obj)
}

// iteratorResult permits protected iterators only as the source of direct range.
func (c *subsetChecks) iteratorResult(call *ast.CallExpr, obj types.Object) {
	if libraryObject(obj) && obj.Name() == "All" && protectedCallable(c.pass.TypesInfo.TypeOf(call)) {
		r, ok := c.unparenthesizedParent(call).(*ast.RangeStmt)
		if !ok || ast.Unparen(r.X) != call {
			c.report(call, "immutable iterators must be consumed by direct range")
		}
	}
}

// unparenthesizedParent finds the context of an expression through parentheses.
func (c *subsetChecks) unparenthesizedParent(expression ast.Expr) ast.Node {
	parent := c.parents[expression]
	for {
		wrapped, ok := parent.(*ast.ParenExpr)
		if !ok {
			return parent
		}
		parent = c.parents[wrapped]
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
	if selection == nil || selection.Kind() == types.FieldVal || !c.protectedRoot(sel.X) {
		return
	}
	if selection.Kind() == types.MethodExpr && libraryObject(selection.Obj()) {
		c.report(sel, "immutable method expressions are unsupported")
		return
	}
	if !c.exactCollectionMethod(sel) {
		c.report(sel, "user-defined methods with immutable storage are unsupported")
		return
	}
	if call, ok := c.unparenthesizedParent(sel).(*ast.CallExpr); ok && ast.Unparen(call.Fun) == sel {
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

// globalStore prevents a local-only reference from entering package or aggregate storage.
func (c *subsetChecks) globalStore(destination, value ast.Expr) {
	if !noRefReference(c.pass.TypesInfo.TypeOf(value)) {
		return
	}
	id, ok := ast.Unparen(destination).(*ast.Ident)
	if !ok {
		c.report(destination, "ImmutNoRef references cannot enter aggregate storage")
		return
	}
	obj := c.pass.TypesInfo.ObjectOf(id)
	if obj != nil && obj.Parent() == c.pass.Pkg.Scope() {
		c.report(destination, "global ImmutNoRef references are unsupported")
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

// callArguments checks typed parameters and the effects of transporting each argument.
func (c *subsetChecks) callArguments(call *ast.CallExpr, obj types.Object, signature *types.Signature, static bool) {
	for i, arg := range call.Args {

		t := c.pass.TypesInfo.TypeOf(arg)
		if !checker.Contains(t) && !protectedCallable(t) {
			continue
		}
		if !static && !libraryObject(obj) {
			c.report(arg, "dynamic calls with immutable storage are unsupported")
		}
		parameter := i
		if signature.Variadic() && parameter >= signature.Params().Len()-1 {
			parameter = signature.Params().Len() - 1
		}
		if parameter >= 0 && parameter < signature.Params().Len() {
			c.transport(arg, signature.Params().At(parameter).Type())
		}
	}
}

// methodCall checks receiver transport and resolves whether dispatch is static.
func (c *subsetChecks) methodCall(call *ast.CallExpr, static bool) bool {
	selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok {
		return static
	}
	selection := c.pass.TypesInfo.Selections[selector]
	if selection == nil || selection.Kind() == types.FieldVal {
		return static
	}
	if _, dynamic := selection.Recv().Underlying().(*types.Interface); dynamic {
		return false
	}
	if !libraryObject(selection.Obj()) && protectedCallable(selection.Obj().Type()) {
		c.report(selector, "user-defined methods with immutable storage are unsupported")
	}
	return static
}
