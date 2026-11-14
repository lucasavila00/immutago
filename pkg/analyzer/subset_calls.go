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

// matchCall accepts exact im APIs, concrete named calls, or calls with only ordinary scalar data.
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
	if matched, rule := c.matchDependencyCall(call, object, signature); matched {
		return rule
	}
	if rule := c.matchCallArguments(call, signature); rule != "" {
		return rule
	}
	if libraryObject(object) {
		return c.matchLibraryCall(call, object)
	}
	if trustedRuntimeCall(object) {
		return ""
	}
	if !c.queries.Callable(signature) && !c.protectedCallArguments(call) {
		return ""
	}
	function, direct := object.(*types.Func)
	if !direct {
		return "dynamic calls with immutable storage are unsupported"
	}
	if selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr); ok {
		if selection := c.pass.TypesInfo.Selections[selector]; selection != nil {
			if _, dynamic := selection.Recv().Underlying().(*types.Interface); dynamic {
				return "dynamic calls with immutable storage are unsupported"
			}
		}
	}
	origin := function.Origin().Type().(*types.Signature)
	if genericSignature(origin) {
		return "user generic instantiations with immutable types are unsupported"
	}
	if origin.Recv() != nil && !c.processSignature(origin) {
		return "user-defined methods with immutable storage are unsupported"
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
		if c.queries.Contains(source) || c.queries.Contains(destination) {
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
		if pointer, ok := c.pass.TypesInfo.TypeOf(call).Underlying().(*types.Pointer); ok && checker.Marked(pointer.Elem()) {
			return ""
		}
	case "make":
		if c.exempt[call] {
			return ""
		}
	case "len", "cap":
		if c.matchBuilderBuiltin(call, builtin.Name()) {
			return ""
		}
		if _, array := c.pass.TypesInfo.TypeOf(call.Args[0]).Underlying().(*types.Array); array {
			return ""
		}
		if !c.queries.Contains(c.pass.TypesInfo.TypeOf(call.Args[0])) {
			return ""
		}
	case "append", "delete", "clear", "copy":
		if c.matchBuilderBuiltin(call, builtin.Name()) {
			signature := c.pass.TypesInfo.TypeOf(call.Fun).Underlying().(*types.Signature)
			return c.matchCallArguments(call, signature)
		}
	}
	if !c.protectedCallArguments(call) && !c.protectedResult(c.pass.TypesInfo.TypeOf(call)) {
		return ""
	}
	return "protected operation is outside the supported expression forms"
}

// matchLibraryCall matches fixed function identities, receivers, and result contexts.
func (c *subsetChecks) matchLibraryCall(call *ast.CallExpr, object types.Object) string {
	switch object.Name() {
	case "Load", "Store":
		if c.ownedSlotCall(call) {
			return ""
		}
		return "Owner requires a direct slot call"
	case "FreezeMap", "FreezeSlice":
		if c.standaloneFreeze(call) {
			return ""
		}
		return "Freeze requires a standalone single binding"
	case "Seal", "Unseal":
		return ""
	case "Get", "At", "Len", "All", "Clone":
		selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
		if !ok || !c.exactCollectionMethod(selector) {
			return "immutable method expressions are unsupported"
		}
		if object.Name() == "All" && c.queries.Callable(c.pass.TypesInfo.TypeOf(call)) {
			statement, ok := c.unparenthesizedParent(call).(*ast.RangeStmt)
			if !ok || ast.Unparen(statement.X) != call {
				return "immutable iterators must be consumed by direct range"
			}
		}
		return ""
	}
	return "protected operation is outside the supported expression forms"
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
	if c.protectedReference(source) && !types.Identical(source, destination) {
		return "reference operation is outside the supported allowlist"
	}
	return ""
}

// protectedCallArguments includes protected tuple components without treating scalar field reads as pointers.
func (c *subsetChecks) protectedCallArguments(call *ast.CallExpr) bool {
	for _, argument := range call.Args {
		typ := c.pass.TypesInfo.TypeOf(argument)
		if c.protectedResult(typ) || c.queries.Callable(typ) || c.ownedExpression(argument) {
			return true
		}
	}
	return false
}

// matchTransport permits concrete value copies and rejects implicit interface or callback storage.
func (c *subsetChecks) matchTransport(source, destination types.Type) string {
	if !c.queries.Contains(source) && !c.queries.Callable(source) {
		return ""
	}
	if destination == nil {
		return ""
	}
	if _, boxed := destination.Underlying().(*types.Interface); boxed {
		return "boxing immutable storage in interfaces is unsupported"
	}
	if c.queries.Callable(source) {
		return "immutable callbacks are unsupported"
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
