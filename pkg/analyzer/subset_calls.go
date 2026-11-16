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

// matchCall accepts fixed im APIs, restricted direct calls, and ordinary calls with persistent values.
func (c *subsetChecks) matchCall(call *ast.CallExpr) string {
	object := c.calledObject(call.Fun)
	if builtin, ok := object.(*types.Builtin); ok {
		return c.matchBuiltin(call, builtin)
	}
	if c.pass.TypesInfo.Types[call.Fun].IsType() {
		return c.matchConversion(call)
	}
	signature, ok := c.pass.TypesInfo.TypeOf(call.Fun).Underlying().(*types.Signature)
	if !ok {
		return "protected operation is outside the supported expression forms"
	}
	if rule := c.matchCallArguments(call, signature); rule != "" {
		return rule
	}
	if libraryObject(object) {
		return c.matchLibraryCall(call, object)
	}
	if runtimeObject(object) {
		return ""
	}
	protected := c.queries.Callable(signature) || c.protectedCallArguments(call)
	if selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr); ok {
		if selection := c.pass.TypesInfo.Selections[selector]; selection != nil && selection.Kind() != types.FieldVal &&
			(protected || c.protectedStorage(selector.X)) {
			return c.matchMethod(selector)
		}
	}
	if !protected {
		return ""
	}
	function, direct := object.(*types.Func)
	if !direct {
		return "dynamic calls require ordinary or persistent Frozen parameters and results"
	}
	origin := function.Origin().Type().(*types.Signature)
	if genericSignature(origin) {
		return "user generic instantiations require ordinary or persistent Frozen type arguments"
	}
	return ""
}

// matchConversion permits ordinary scalar conversions and checks both protected endpoints.
func (c *subsetChecks) matchConversion(call *ast.CallExpr) string {

	if c.protectedReference(c.pass.TypesInfo.TypeOf(call)) {
		return "shared pointer conversions are unsupported"
	}
	if len(call.Args) == 1 {
		source, destination := c.pass.TypesInfo.TypeOf(call.Args[0]), c.pass.TypesInfo.TypeOf(call)
		if rule := c.matchTransport(source, destination); rule != "" {
			return rule
		}
		if checker.FrozenInterface(destination) && source != types.Typ[types.UntypedNil] {
			return ""
		}
		if _, boxed := destination.Underlying().(*types.Interface); boxed &&
			!checker.FrozenInterface(destination) && !c.queries.Transport(source) {
			return ""
		}
		if c.queries.Owns(source) || c.queries.Owns(destination) {
			return "protected value conversions are unsupported"
		}
		return ""
	}
	return "protected operation is outside the supported expression forms"
}

// matchBuiltin lists the documented allocation and native builder operations.
func (c *subsetChecks) matchBuiltin(call *ast.CallExpr, builtin *types.Builtin) string {
	switch builtin.Name() {
	case "new":
		if pointer, ok := c.pass.TypesInfo.TypeOf(call).Underlying().(*types.Pointer); ok &&
			(checker.Marked(pointer.Elem()) || !c.queries.Owns(pointer.Elem())) {
			return ""
		}
	case "make":
		if c.exempt[call] || !c.queries.Owns(c.pass.TypesInfo.TypeOf(call)) {
			return ""
		}
	case "len", "cap":
		if c.matchBuilderBuiltin(call, builtin.Name()) {
			return ""
		}
		if _, array := c.pass.TypesInfo.TypeOf(call.Args[0]).Underlying().(*types.Array); array {
			return ""
		}
		if !c.queries.Owns(c.pass.TypesInfo.TypeOf(call.Args[0])) {
			return ""
		}
	case "append", "delete", "clear", "copy":
		if c.matchBuilderBuiltin(call, builtin.Name()) || !c.protectedStorage(call.Args[0]) {
			signature := c.pass.TypesInfo.TypeOf(call.Fun).Underlying().(*types.Signature)
			return c.matchCallArguments(call, signature)
		}
	}
	if !c.protectedCallArguments(call) && !c.queries.Owns(c.pass.TypesInfo.TypeOf(call)) {
		return ""
	}
	return "protected operation is outside the supported expression forms"
}

// matchLibraryCall matches fixed function identities, receivers, and result contexts.
func (c *subsetChecks) matchLibraryCall(call *ast.CallExpr, object types.Object) string {
	switch object.Name() {
	case "Load", "LoadPointer", "Store":
		if c.ownedSlotCall(call) {
			return ""
		}
		return "Cell requires a direct slot call"
	case "FreezeMap", "FreezeSlice":
		if c.standaloneFreeze(call) {
			return ""
		}
		return "Freeze requires a standalone single binding"
	case "Trust", "Unwrap", "ProjectionOf", "CopyRecord", "FreezeString", "ProjectString", "IsNil",
		"StringOf", "StringOfFrozen", "SliceOf", "MapOf":
		return ""
	case "Value":
		typ := c.pass.TypesInfo.TypeOf(call)
		if reason := c.proofs.FrozenReason(typ); reason != "" {
			return reason
		}
		return c.proofs.ReadResultReason(typ)
	case "Get", "At", "AtPointer", "Len", "All", "Clone", "Equal", "EqualString", "Is", "Unsafe", "Projection":
		return c.matchLibraryMethod(call, object.Name())
	}
	return "protected operation is outside the supported expression forms"
}

// matchLibraryMethod checks the sealed receiver and any result storage restriction.
func (c *subsetChecks) matchLibraryMethod(call *ast.CallExpr, method string) string {
	selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok || !c.exactCollectionMethod(selector) {
		return "method expressions require ordinary or persistent Frozen receivers and contracts"
	}
	receiver := c.pass.TypesInfo.TypeOf(selector.X)
	if checker.IsProjection(receiver) && method == "Clone" {
		args := types.Unalias(receiver).(*types.Named).TypeArgs()
		for index := 0; index < args.Len(); index++ {
			if reason := c.proofs.CopiedReason(args.At(index)); reason != "" {
				return reason
			}
		}
	}
	if method == "All" && (checker.IsProjection(receiver) || c.queries.Callable(c.pass.TypesInfo.TypeOf(call))) {
		statement, ok := c.unparenthesizedParent(call).(*ast.RangeStmt)
		if !ok || ast.Unparen(statement.X) != call {
			return "immutable iterators must be consumed by direct range"
		}
	}
	return ""
}

// matchCallArguments resolves every actual parameter, including forwarded tuples and variadic elements.
func (c *subsetChecks) matchCallArguments(call *ast.CallExpr, signature *types.Signature) string {
	for index, argument := range call.Args {
		source := c.pass.TypesInfo.TypeOf(argument)
		if tuple, ok := source.(*types.Tuple); ok {
			for item := 0; item < tuple.Len(); item++ {
				if rule := c.matchArgument(tuple.At(item).Type(), callParameter(signature, item, false)); rule != "" {
					return rule
				}
			}
		} else if rule := c.matchArgument(source, callParameter(signature, index, call.Ellipsis.IsValid())); rule != "" {
			return rule
		}
	}
	return ""
}

// matchArgument permits protected pointers only with the exact concrete parameter type.
func (c *subsetChecks) matchArgument(source, destination types.Type) string {
	if rule := c.matchTransport(source, destination); rule != "" {
		return rule
	}
	if c.protectedReference(source) && !types.Identical(source, destination) && !checker.FrozenInterface(destination) {
		return "reference operation is outside the supported allowlist"
	}
	return ""
}

// protectedCallArguments includes protected tuple components without treating scalar field reads as pointers.
func (c *subsetChecks) protectedCallArguments(call *ast.CallExpr) bool {
	for _, argument := range call.Args {
		typ := c.pass.TypesInfo.TypeOf(argument)
		if c.queries.Transport(typ) || c.queries.Callable(typ) || c.ownedExpression(argument) {
			return true
		}
	}
	return false
}

// matchTransport proves Frozen interface implementations and rejects restricted value erasure.
func (c *subsetChecks) matchTransport(source, destination types.Type) string {
	if checker.FrozenInterface(destination) {
		return c.matchFrozenInterface(source, destination)
	}
	if !c.queries.Transport(source) && !c.queries.Callable(source) {
		return ""
	}
	if destination == nil {
		return ""
	}
	if _, boxed := destination.Underlying().(*types.Interface); boxed {
		return "ordinary interfaces require ordinary or persistent Frozen values"
	}
	if c.queries.Callable(source) {
		return "callbacks require ordinary or persistent Frozen parameters and results"
	}
	return ""
}

// unparenthesizedParent finds the consuming operation through parentheses.
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

// directCall follows only parentheses and type arguments to a direct callee.
func (c *subsetChecks) directCall(node ast.Node) *ast.CallExpr {
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

// returnType finds the declared result type for implicit boxing checks.
func (c *subsetChecks) returnType(statement *ast.ReturnStmt, index int) types.Type {
	results := c.functionSignature(statement).Results()
	if index < results.Len() {
		return results.At(index).Type()
	}
	return nil
}

// functionSignature resolves the compiler contract of the enclosing function.
func (c *subsetChecks) functionSignature(node ast.Node) *types.Signature {
	for parent := c.parents[node]; parent != nil; parent = c.parents[parent] {
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
		return t.Underlying().(*types.Signature)
	}
	return nil
}

// genericSignature includes receiver parameters, which method instantiation retains separately.
func genericSignature(signature *types.Signature) bool {
	return signature.TypeParams().Len() > 0 || signature.RecvTypeParams().Len() > 0
}

// callParameter resolves the destination of one argument, including a variadic element.
func callParameter(signature *types.Signature, index int, expanded bool) types.Type {
	parameters := signature.Params()
	if signature.Variadic() && index >= parameters.Len()-1 {
		parameter := parameters.At(parameters.Len() - 1).Type()
		if !expanded {
			return parameter.Underlying().(*types.Slice).Elem()
		}
		return parameter
	}
	if index < parameters.Len() {
		return parameters.At(index).Type()
	}
	return nil
}
