package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// serviceType identifies a named struct that embeds supervise.Service, with one result per type.
func (c *subsetChecks) serviceType(t types.Type) bool {
	if t == nil {
		return false
	}
	if c.services == nil {
		c.services = make(map[types.Type]bool)
	}
	result, ok := c.services[t]
	if !ok {
		result = serviceDeclaration(t)
		c.services[t] = result
	}
	return result
}

// serviceDeclaration identifies a named struct that embeds supervise.Service.
func serviceDeclaration(t types.Type) bool {
	_, named := types.Unalias(t).(*types.Named)
	return named && checker.Service(t)
}

// serviceName returns the type name of a service or a service pointer.
func serviceName(t types.Type) string {
	if pointer, ok := t.Underlying().(*types.Pointer); ok {
		t = pointer.Elem()
	}
	return types.Unalias(t).(*types.Named).Obj().Name()
}

// serviceValue identifies a service or a pointer to one.
func (c *subsetChecks) serviceValue(t types.Type) bool {
	if pointer, ok := t.Underlying().(*types.Pointer); ok {
		return c.serviceType(pointer.Elem())
	}
	return c.serviceType(t)
}

// serviceSignature identifies a method whose receiver is a service pointer.
func (c *subsetChecks) serviceSignature(signature *types.Signature) bool {
	return signature.Recv() != nil && c.serviceValue(signature.Recv().Type())
}

// runtimeObject identifies a supervise object by package identity and name.
func runtimeObject(object types.Object, name string) bool {
	return object != nil && object.Pkg() != nil && object.Pkg().Path() == checker.RuntimePath && object.Name() == name
}

// serviceTypeArgument identifies the supervise identities whose first type argument is a service.
func serviceTypeArgument(object types.Object) bool {
	return runtimeObject(object, checker.RefType) || runtimeObject(object, checker.StartFunction) ||
		runtimeObject(object, checker.ReserveFunction) || runtimeObject(object, checker.StartReservedFunction)
}

// trustedRuntimeCall identifies the supervise calls that take or return service arguments.
func trustedRuntimeCall(object types.Object) bool {
	function, ok := object.(*types.Func)
	if !ok {
		return false
	}
	signature := function.Type().(*types.Signature)
	if signature.Recv() == nil {
		return runtimeObject(object, checker.StartFunction) || runtimeObject(object, checker.StartReservedFunction)
	}
	return runtimeMethod(object, checker.ServiceMarker, checker.ArgsMethod)
}

// currentMethod identifies the supervise.Ref.Current method.
func currentMethod(object types.Object) bool {
	return runtimeMethod(object, checker.RefType, checker.CurrentMethod)
}

// runtimeMethod checks the declared receiver identity before granting a runtime method its special use.
func runtimeMethod(object types.Object, receiver, name string) bool {
	function, ok := object.(*types.Func)
	if !ok || !runtimeObject(function, name) {
		return false
	}
	signature := function.Type().(*types.Signature)
	if signature.Recv() == nil {
		return false
	}
	t := signature.Recv().Type()
	if pointer, ok := t.(*types.Pointer); ok {
		t = pointer.Elem()
	}
	return checker.IsRuntimeType(t, receiver)
}

// serviceMarker identifies the exact supervise.Service marker type.
func serviceMarker(t types.Type) bool { return checker.IsRuntimeType(t, checker.ServiceMarker) }

const serviceEscape = "service pointer escapes; use ref.Current() only as the receiver of a direct exported method call"

const runChildrenRule = "call Children() directly on the receiver of a service method, " +
	"and pass the result only as the parent of supervise.Start, supervise.StartReserved, or supervise.StartGroup, " +
	"directly or through one local variable"

// checkServiceNode checks service declarations, service type names, and service values.
func (c *subsetChecks) checkServiceNode(node ast.Node) {
	switch n := node.(type) {
	case *ast.StructType:
		c.checkServiceStruct(n)
	case *ast.Ident:
		c.checkServiceName(n)
	case *ast.SelectorExpr:
		object := c.pass.TypesInfo.ObjectOf(n.Sel)
		if c.runtimeEntryPoint(object) {
			c.pass.Reportf(n.Pos(), "only the runtime can call a service %s method", object.Name())
		}
		if currentMethod(object) && !c.directCurrent(n) {
			c.pass.Reportf(n.Pos(), serviceEscape)
		}
		if runtimeMethod(object, checker.ServiceMarker, checker.ChildrenMethod) && !c.runChildren(n) {
			c.pass.Reportf(n.Pos(), runChildrenRule)
		}
		if runtimeMethod(object, checker.ConcurrentSafeType, checker.ValueMethod) && !c.dependencyValue(n) {
			c.pass.Reportf(n.Pos(), "call Value only directly on l.Args() or on a field of l.Args() in a service method")
		}
	}
	if expression, ok := node.(ast.Expr); ok {
		c.checkServiceExpression(expression)
	}
}

// dependencyValue permits a direct synchronous supervise.ConcurrentSafe.Value call
// on the arguments of the service receiver or on one of their fields.
func (c *subsetChecks) dependencyValue(selector *ast.SelectorExpr) bool {
	if !runtimeMethod(c.pass.TypesInfo.ObjectOf(selector.Sel), checker.ConcurrentSafeType, checker.ValueMethod) ||
		!c.synchronousCall(selector) {
		return false
	}
	source := ast.Unparen(selector.X)
	if field, ok := source.(*ast.SelectorExpr); ok {
		selection := c.pass.TypesInfo.Selections[field]
		if selection == nil || selection.Kind() != types.FieldVal || len(selection.Index()) != 1 {
			return false
		}
		source = ast.Unparen(field.X)
	}
	call, ok := source.(*ast.CallExpr)
	if !ok || len(call.Args) != 0 {
		return false
	}
	args, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok || !runtimeMethod(c.pass.TypesInfo.ObjectOf(args.Sel), checker.ServiceMarker, checker.ArgsMethod) {
		return false
	}
	receiver, ok := ast.Unparen(args.X).(*ast.Ident)
	variable, _ := c.pass.TypesInfo.Uses[receiver].(*types.Var)
	return ok && variable != nil && variable.Kind() == types.RecvVar && c.serviceValue(variable.Type())
}

// checkServiceStruct requires one embedded supervise.Service in a defined, non-generic struct without a marker.
func (c *subsetChecks) checkServiceStruct(n *ast.StructType) {
	t := c.pass.TypesInfo.TypeOf(n)
	structure, _ := t.Underlying().(*types.Struct)
	if structure == nil {
		return
	}
	count := 0
	for field := range structure.Fields() {
		if field.Embedded() && serviceMarker(field.Type()) {
			count++
			continue
		}
		pointer, ok := field.Type().Underlying().(*types.Pointer)
		if serviceMarker(field.Type()) || ok && serviceMarker(pointer.Elem()) {
			c.pass.Reportf(n.Pos(), "supervise.Service must be embedded by value")
		}
	}
	if count == 0 {
		return
	}
	spec, defined := c.parents[n].(*ast.TypeSpec)
	switch {
	case !defined || spec.Assign.IsValid():
		c.pass.Reportf(n.Pos(), "services require a defined struct type")
	case spec.TypeParams != nil:
		c.pass.Reportf(n.Pos(), "services cannot be generic")
	case count > 1 || checker.Marked(t):
		c.pass.Reportf(n.Pos(), "services embed supervise.Service once and no immutable marker")
	case !serviceRun(c.pass.TypesInfo.Defs[spec.Name].Type()):
		c.pass.Reportf(n.Pos(), "a service must declare Run(context.Context) with a pointer receiver and no results")
	case !serviceInit(c.pass.TypesInfo.Defs[spec.Name].Type()):
		c.pass.Reportf(n.Pos(), "a service must use the default Init or declare Init() error with a pointer receiver")
	}
}

// serviceRun checks the service entry point without accepting a promoted method or an unrelated Run signature.
func serviceRun(t types.Type) bool {
	object, index, _ := types.LookupFieldOrMethod(types.NewPointer(t), false, nil, checker.RunMethod)
	method, ok := object.(*types.Func)
	if !ok || len(index) != 1 {
		return false
	}
	signature := method.Type().(*types.Signature)
	if signature.Variadic() || signature.Params().Len() != 1 || signature.Results().Len() != 0 {
		return false
	}
	pointer, ok := signature.Recv().Type().(*types.Pointer)
	if !ok || !types.Identical(pointer.Elem(), t) {
		return false
	}
	parameter, ok := types.Unalias(signature.Params().At(0).Type()).(*types.Named)
	return ok && parameter.Obj().Pkg() != nil && parameter.Obj().Pkg().Path() == "context" &&
		parameter.Obj().Name() == "Context"
}

// serviceInit accepts the exact default method or a directly declared Init() error.
func serviceInit(t types.Type) bool {
	object, index, _ := types.LookupFieldOrMethod(types.NewPointer(t), false, nil, checker.InitMethod)
	method, ok := object.(*types.Func)
	if !ok {
		return false
	}
	if len(index) == 2 && runtimeMethod(method, checker.ServiceMarker, checker.InitMethod) {
		return true
	}
	if len(index) != 1 {
		return false
	}
	signature := method.Type().(*types.Signature)
	pointer, ok := signature.Recv().Type().(*types.Pointer)
	return ok && types.Identical(pointer.Elem(), t) && !genericSignature(signature) &&
		!signature.Variadic() && signature.Params().Len() == 0 && signature.Results().Len() == 1 &&
		types.Identical(signature.Results().At(0).Type(), types.Universe.Lookup("error").Type())
}

// serviceEntryPoint identifies methods called only by the runtime.
func serviceEntryPoint(name string) bool {
	return name == checker.RunMethod || name == checker.InitMethod
}

// runtimeEntryPoint identifies Run on a service and explicit or default Init methods.
func (c *subsetChecks) runtimeEntryPoint(object types.Object) bool {
	method, ok := object.(*types.Func)
	return ok && serviceEntryPoint(method.Name()) &&
		(c.serviceSignature(method.Type().(*types.Signature)) ||
			runtimeMethod(method, checker.ServiceMarker, checker.InitMethod))
}

// checkServiceName permits a service type name only as a pointer receiver or a supervise type argument.
func (c *subsetChecks) checkServiceName(id *ast.Ident) {
	name, ok := c.pass.TypesInfo.Uses[id].(*types.TypeName)
	if !ok || !c.serviceType(name.Type()) {
		return
	}
	var expression ast.Expr = id
	if selector, ok := c.parents[id].(*ast.SelectorExpr); ok && selector.Sel == id {
		expression = selector
	}
	switch parent := c.unparenthesizedParent(expression).(type) {
	case *ast.StarExpr:
		if c.serviceReceiver(parent) {
			return
		}
	case *ast.IndexExpr:
		object := c.calledObject(parent.X)
		if ast.Unparen(parent.Index) == expression && serviceTypeArgument(object) {
			return
		}
	}
	c.pass.Reportf(id.Pos(), "service type %[1]s can be named only in its declaration, a pointer receiver, "+
		"supervise.Start[%[1]s], supervise.StartReserved[%[1]s], supervise.Reserve[%[1]s], or supervise.Ref[%[1]s]", id.Name)
}

// serviceReceiver identifies the type of a method receiver.
func (c *subsetChecks) serviceReceiver(receiver *ast.StarExpr) bool {
	field, ok := c.unparenthesizedParent(receiver).(*ast.Field)
	if !ok {
		return false
	}
	list := c.parents[field]
	declaration, ok := c.parents[list].(*ast.FuncDecl)
	return ok && declaration.Recv == list
}

// checkServiceExpression permits service values only as a method receiver or a direct Ref.Current result.
func (c *subsetChecks) checkServiceExpression(expression ast.Expr) {
	if _, paren := expression.(*ast.ParenExpr); paren || c.pass.TypesInfo.Types[expression].IsType() {
		return
	}
	id, name := expression.(*ast.Ident)
	if _, typeName := c.pass.TypesInfo.Uses[id].(*types.TypeName); name && (typeName || c.pass.TypesInfo.Defs[id] != nil) {
		return
	}
	t := c.pass.TypesInfo.TypeOf(expression)
	if t == nil || !c.serviceValue(t) {
		return
	}
	if variable, ok := c.pass.TypesInfo.Uses[id].(*types.Var); name && ok && variable.Kind() == types.RecvVar {
		if c.insideClosure(id) {
			c.pass.Reportf(id.Pos(), "closures cannot capture a service receiver")
		} else if !c.allowedReceiverUse(id) {
			c.pass.Reportf(id.Pos(), "service receivers can only select a field or call a method directly")
		}
		return
	}
	if call, ok := expression.(*ast.CallExpr); ok && currentMethod(c.calledObject(call.Fun)) {
		if !c.allowedCurrentUse(call) {
			c.pass.Reportf(call.Pos(), serviceEscape)
		}
		return
	}
	if c.serviceCreation(expression) {
		c.pass.Reportf(expression.Pos(), "service %[1]s is created outside supervise.Start; use supervise.Start[%[1]s](parent, args)",
			serviceName(t))
		return
	}
	c.pass.Reportf(expression.Pos(), serviceEscape)
}

// serviceCreation identifies a composite literal or a new call.
func (c *subsetChecks) serviceCreation(expression ast.Expr) bool {
	switch creation := expression.(type) {
	case *ast.CompositeLit:
		return true
	case *ast.CallExpr:
		builtin, ok := c.calledObject(creation.Fun).(*types.Builtin)
		return ok && builtin.Name() == "new"
	}
	return false
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

// allowedReceiverUse permits fields other than the marker and direct synchronous calls other than Run or Init.
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
		return !serviceMarker(selection.Type())
	}
	return selection.Kind() == types.MethodVal && !serviceEntryPoint(selection.Obj().Name()) &&
		c.synchronousCall(selector)
}

// allowedCurrentUse permits a direct synchronous call of an exported method declared on the service.
func (c *subsetChecks) allowedCurrentUse(current *ast.CallExpr) bool {
	selector, ok := c.unparenthesizedParent(current).(*ast.SelectorExpr)
	if !ok || ast.Unparen(selector.X) != current {
		return false
	}
	selection := c.pass.TypesInfo.Selections[selector]
	return selection != nil && selection.Kind() == types.MethodVal && len(selection.Index()) == 1 &&
		selection.Obj().Exported() && !serviceEntryPoint(selection.Obj().Name()) && c.synchronousCall(selector)
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

// runChildren permits a direct synchronous Group call on a service receiver whose result is a parent
// argument, directly or through one local variable that has no other use.
func (c *subsetChecks) runChildren(selector *ast.SelectorExpr) bool {
	receiver, _ := ast.Unparen(selector.X).(*ast.Ident)
	variable, ok := c.pass.TypesInfo.Uses[receiver].(*types.Var)
	if !ok || variable.Kind() != types.RecvVar || c.insideClosure(selector) || !c.synchronousCall(selector) {
		return false
	}
	call := c.directCall(selector)
	if c.parentArgument(call) {
		return true
	}
	assignment, ok := c.unparenthesizedParent(call).(*ast.AssignStmt)
	if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
		return false
	}
	name, _ := assignment.Lhs[0].(*ast.Ident)
	local, ok := c.pass.TypesInfo.Defs[name].(*types.Var)
	if !ok {
		return false
	}
	for use, object := range c.pass.TypesInfo.Uses {
		if object == local && (c.insideClosure(use) || !c.parentArgument(use)) {
			return false
		}
	}
	return true
}

// parentArgument identifies the first argument of a direct synchronous supervise.Start, supervise.StartReserved, or
// supervise.StartGroup call.
func (c *subsetChecks) parentArgument(expression ast.Expr) bool {
	call, ok := c.unparenthesizedParent(expression).(*ast.CallExpr)
	if !ok || len(call.Args) == 0 || ast.Unparen(call.Args[0]) != expression {
		return false
	}
	object := c.calledObject(call.Fun)
	if !runtimeObject(object, checker.StartFunction) && !runtimeObject(object, checker.StartReservedFunction) &&
		!runtimeObject(object, checker.StartGroupFunction) {
		return false
	}
	switch c.parents[call].(type) {
	case *ast.GoStmt, *ast.DeferStmt:
		return false
	}
	return true
}

// directCurrent identifies a Ref.Current method selection that is called directly.
func (c *subsetChecks) directCurrent(selector *ast.SelectorExpr) bool {
	selection := c.pass.TypesInfo.Selections[selector]
	return selection != nil && selection.Kind() == types.MethodVal && c.directCall(selector) != nil
}

// checkServiceMethod requires allowed stored types in each parameter and result of an exported service method.
func (c *checks) checkServiceMethod(declaration *ast.FuncDecl) {
	if declaration.Recv == nil || !declaration.Name.IsExported() || serviceEntryPoint(declaration.Name.Name) {
		return
	}
	signature := c.pass.TypesInfo.Defs[declaration.Name].Type().(*types.Signature)
	receiver := signature.Recv().Type()
	if pointer, ok := receiver.Underlying().(*types.Pointer); ok {
		receiver = pointer.Elem()
	}
	if !checker.Service(receiver) {
		return
	}
	for _, tuple := range []*types.Tuple{signature.Params(), signature.Results()} {
		for variable := range tuple.Variables() {
			if reason := c.proofs.Reason(variable.Type()); reason != "" {
				c.report(declaration.Name.Pos(), "exported service method "+declaration.Name.Name+" carries "+
					diagnosticType(variable.Type())+", which is not an allowed stored type: "+reason)
			}
		}
	}
}

// checkRuntimeArguments requires a service type argument for supervise identities and allowed service arguments.
func (c *checks) checkRuntimeArguments(ident *ast.Ident, obj types.Object, instance types.Instance) {
	argument := instance.TypeArgs.At(0)
	if serviceTypeArgument(obj) && !serviceDeclaration(argument) {
		c.report(ident.Pos(), "supervise.Start, supervise.StartReserved, supervise.Reserve, and supervise.Ref need a service type argument")
	}
	if runtimeObject(obj, checker.ServiceMarker) {
		if reason := c.proofs.ArgumentReason(argument); reason != "" {
			c.report(ident.Pos(), "service arguments "+diagnosticType(argument)+" are not an allowed stored type: "+reason)
		}
	}
}
