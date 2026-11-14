package analyzer

import (
	"go/ast"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// processType identifies a named struct that embeds im.Process, with one result per type.
func (c *subsetChecks) processType(t types.Type) bool {
	if t == nil {
		return false
	}
	if c.processes == nil {
		c.processes = make(map[types.Type]bool)
	}
	result, ok := c.processes[t]
	if !ok {
		result = processDeclaration(t)
		c.processes[t] = result
	}
	return result
}

// processDeclaration identifies a named struct that embeds im.Process.
func processDeclaration(t types.Type) bool {
	_, named := types.Unalias(t).(*types.Named)
	return named && checker.Process(t)
}

// processValue identifies a process or a pointer to one.
func (c *subsetChecks) processValue(t types.Type) bool {
	if pointer, ok := t.Underlying().(*types.Pointer); ok {
		return c.processType(pointer.Elem())
	}
	return c.processType(t)
}

// processSignature identifies a method whose receiver is a process pointer.
func (c *subsetChecks) processSignature(signature *types.Signature) bool {
	return signature.Recv() != nil && c.processValue(signature.Recv().Type())
}

// runtimeObject identifies a proc object by package identity and name.
func runtimeObject(object types.Object, name string) bool {
	return object != nil && object.Pkg() != nil && object.Pkg().Path() == checker.RuntimePath && object.Name() == name
}

// processTypeArgument identifies proc.Ref and proc.Start, which take a process type argument.
func processTypeArgument(object types.Object) bool {
	return runtimeObject(object, checker.RefType) || runtimeObject(object, checker.StartFunction)
}

// refGet identifies the proc.Ref.Get method.
func refGet(object types.Object) bool { return runtimeObject(object, checker.GetMethod) }

// processMarker identifies the exact im.Process marker type.
func processMarker(t types.Type) bool { return checker.IsLibraryType(t, checker.ProcessMarker) }

// checkProcessNode checks process declarations, process type names, and process values.
func (c *subsetChecks) checkProcessNode(node ast.Node) {
	switch n := node.(type) {
	case *ast.StructType:
		c.checkProcessStruct(n)
	case *ast.Ident:
		c.checkProcessName(n)
	case *ast.SelectorExpr:
		if refGet(c.pass.TypesInfo.ObjectOf(n.Sel)) && !c.directRefGet(n) {
			c.pass.Reportf(n.Pos(), "proc.Ref.Get requires a direct call")
		}
	}
	if expression, ok := node.(ast.Expr); ok {
		c.checkProcessExpression(expression)
	}
}

// checkProcessStruct requires one embedded im.Process in a defined, non-generic struct without a marker.
func (c *subsetChecks) checkProcessStruct(n *ast.StructType) {
	t := c.pass.TypesInfo.TypeOf(n)
	structure, _ := t.Underlying().(*types.Struct)
	if structure == nil {
		return
	}
	count := 0
	for field := range structure.Fields() {
		if field.Embedded() && processMarker(field.Type()) {
			count++
			continue
		}
		pointer, ok := field.Type().Underlying().(*types.Pointer)
		if processMarker(field.Type()) || ok && processMarker(pointer.Elem()) {
			c.pass.Reportf(n.Pos(), "im.Process must be embedded by value")
		}
	}
	if count == 0 {
		return
	}
	spec, defined := c.parents[n].(*ast.TypeSpec)
	switch {
	case !defined || spec.Assign.IsValid():
		c.pass.Reportf(n.Pos(), "processes require a defined struct type")
	case spec.TypeParams != nil:
		c.pass.Reportf(n.Pos(), "processes cannot be generic")
	case count > 1 || checker.Marked(t):
		c.pass.Reportf(n.Pos(), "processes embed im.Process once and no immutable marker")
	}
}

// checkProcessName permits a process type name only as a pointer receiver or a proc type argument.
func (c *subsetChecks) checkProcessName(id *ast.Ident) {
	name, ok := c.pass.TypesInfo.Uses[id].(*types.TypeName)
	if !ok || !c.processType(name.Type()) {
		return
	}
	var expression ast.Expr = id
	if selector, ok := c.parents[id].(*ast.SelectorExpr); ok && selector.Sel == id {
		expression = selector
	}
	switch parent := c.unparenthesizedParent(expression).(type) {
	case *ast.StarExpr:
		if c.processReceiver(parent) {
			return
		}
	case *ast.IndexExpr:
		object := c.calledObject(parent.X)
		if ast.Unparen(parent.Index) == expression && processTypeArgument(object) {
			return
		}
	}
	c.pass.Reportf(id.Pos(), "process types can be named only in their declaration, a pointer receiver, "+
		"proc.Start, or proc.Ref")
}

// processReceiver identifies the type of a method receiver.
func (c *subsetChecks) processReceiver(receiver *ast.StarExpr) bool {
	field, ok := c.unparenthesizedParent(receiver).(*ast.Field)
	if !ok {
		return false
	}
	list := c.parents[field]
	declaration, ok := c.parents[list].(*ast.FuncDecl)
	return ok && declaration.Recv == list
}

// checkProcessExpression permits process values only as a method receiver or a direct Ref.Get result.
func (c *subsetChecks) checkProcessExpression(expression ast.Expr) {
	if _, paren := expression.(*ast.ParenExpr); paren || c.pass.TypesInfo.Types[expression].IsType() {
		return
	}
	id, name := expression.(*ast.Ident)
	if name && c.pass.TypesInfo.Defs[id] != nil {
		return
	}
	t := c.pass.TypesInfo.TypeOf(expression)
	if t == nil || !c.processValue(t) {
		return
	}
	if variable, ok := c.pass.TypesInfo.Uses[id].(*types.Var); name && ok && variable.Kind() == types.RecvVar {
		if c.insideClosure(id) {
			c.pass.Reportf(id.Pos(), "closures cannot capture a process receiver")
		} else if !c.allowedReceiverUse(id) {
			c.pass.Reportf(id.Pos(), "process receivers can only select a field or call a method directly")
		}
		return
	}
	if call, ok := expression.(*ast.CallExpr); ok && refGet(c.calledObject(call.Fun)) {
		if !c.allowedGetUse(call) {
			c.pass.Reportf(call.Pos(), "a Ref.Get result can only receive a direct call of an exported method other than Run")
		}
		return
	}
	c.pass.Reportf(expression.Pos(), "process values can only come from a method receiver or Ref.Get")
}

// insideClosure reports a use in a function literal within the method.
func (c *subsetChecks) insideClosure(node ast.Node) bool {
	for node != nil {
		switch node.(type) {
		case *ast.FuncLit:
			return true
		case *ast.FuncDecl:
			return false
		}
		node = c.parents[node]
	}
	return false
}

// allowedReceiverUse permits field selection and a direct synchronous method call other than Run.
func (c *subsetChecks) allowedReceiverUse(receiver *ast.Ident) bool {
	selector, ok := c.unparenthesizedParent(receiver).(*ast.SelectorExpr)
	if !ok || ast.Unparen(selector.X) != receiver {
		return false
	}
	selection := c.pass.TypesInfo.Selections[selector]
	if selection == nil {
		return false
	}
	if selection.Kind() == types.FieldVal {
		return true
	}
	return selection.Kind() == types.MethodVal && selection.Obj().Name() != "Run" && c.synchronousCall(selector)
}

// allowedGetUse permits a direct synchronous call of an exported method declared on the process.
func (c *subsetChecks) allowedGetUse(get *ast.CallExpr) bool {
	selector, ok := c.unparenthesizedParent(get).(*ast.SelectorExpr)
	if !ok || ast.Unparen(selector.X) != get {
		return false
	}
	selection := c.pass.TypesInfo.Selections[selector]
	return selection != nil && selection.Kind() == types.MethodVal && len(selection.Index()) == 1 &&
		selection.Obj().Exported() && selection.Obj().Name() != "Run" && c.synchronousCall(selector)
}

// synchronousCall requires a direct call that is not a go or defer statement.
func (c *subsetChecks) synchronousCall(selector *ast.SelectorExpr) bool {
	call := c.directCall(selector)
	if call == nil {
		return false
	}
	switch c.parents[call].(type) {
	case *ast.GoStmt, *ast.DeferStmt:
		return false
	}
	return true
}

// directRefGet identifies a Ref.Get method selection that is called directly.
func (c *subsetChecks) directRefGet(selector *ast.SelectorExpr) bool {
	selection := c.pass.TypesInfo.Selections[selector]
	return selection != nil && selection.Kind() == types.MethodVal && c.directCall(selector) != nil
}

// checkProcessMethod requires allowed stored types in each parameter and result of an exported process method.
func (c *checks) checkProcessMethod(declaration *ast.FuncDecl) {
	if declaration.Recv == nil || !declaration.Name.IsExported() || declaration.Name.Name == "Run" {
		return
	}
	signature := c.pass.TypesInfo.Defs[declaration.Name].Type().(*types.Signature)
	receiver := signature.Recv().Type()
	if pointer, ok := receiver.Underlying().(*types.Pointer); ok {
		receiver = pointer.Elem()
	}
	if !checker.Process(receiver) {
		return
	}
	for _, tuple := range []*types.Tuple{signature.Params(), signature.Results()} {
		for variable := range tuple.Variables() {
			if reason := c.proofs.Reason(variable.Type()); reason != "" {
				c.report(declaration.Name.Pos(), "exported process method "+declaration.Name.Name+" carries "+
					diagnosticType(variable.Type())+", which is not an allowed stored type: "+reason)
			}
		}
	}
}
