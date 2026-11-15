package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// declarationNode checks contracts that have no runtime operand, plus closure scopes.
func (c *subsetChecks) declarationNode(node ast.Node) {
	switch declaration := node.(type) {
	case *ast.FuncDecl:
		c.functionDeclaration(declaration)
	case *ast.TypeSpec:
		if declaration.TypeParams != nil {
			c.genericStorage(declaration, declaration.Name)
		}
	case *ast.ValueSpec:
		c.valueDeclaration(declaration)
	case *ast.AssignStmt:
		c.assignmentBinding(declaration)
	case *ast.TypeSwitchStmt:
		c.typeSwitchDeclaration(declaration)
	case *ast.FuncLit:
		c.matchClosure(declaration)
	}
}

// assignmentBinding activates complete binding checks for tracked short-declaration destinations.
func (c *subsetChecks) assignmentBinding(statement *ast.AssignStmt) {
	for _, destination := range statement.Lhs {
		if c.protectedStorage(destination) {
			if rule := c.matchAssignment(statement); rule != "" {
				c.report(statement, rule)
			}
			return
		}
	}
}

// functionDeclaration checks the exact contract for named functions and methods.
func (c *subsetChecks) functionDeclaration(declaration *ast.FuncDecl) {
	signature := c.pass.TypesInfo.Defs[declaration.Name].Type().(*types.Signature)
	if rule := subsetResults(signature.Results()); rule != "" {
		c.report(declaration.Name, rule)
	}
	if genericSignature(signature) {
		c.genericStorage(declaration, declaration.Name)
	}
	if signature.Recv() != nil && c.queries.Contains(signature.Recv().Type()) && !c.markedMethodReceiver(signature) {
		c.report(declaration.Name, "immutable methods require a concrete non-generic marked receiver")
	}
}

// valueDeclaration activates binding checks even when the initializer is absent or nil.
func (c *subsetChecks) valueDeclaration(declaration *ast.ValueSpec) {
	for _, name := range declaration.Names {
		variable := c.pass.TypesInfo.Defs[name]
		if variable != nil && (c.queries.Contains(variable.Type()) || c.ownedExpression(name)) {
			if rule := c.matchDeclaration(declaration); rule != "" {
				c.report(declaration, rule)
			}
			break
		}
	}
}

// typeSwitchDeclaration rejects interface recovery through a protected case type.
func (c *subsetChecks) typeSwitchDeclaration(declaration *ast.TypeSwitchStmt) {
	for _, node := range declaration.Body.List {
		for _, expression := range node.(*ast.CaseClause).List {
			if c.queries.Contains(c.pass.TypesInfo.TypeOf(expression)) {
				c.report(expression, "protected storage is unsupported in type switches")
			}
		}
	}
}

// matchAssignment resolves the whole binding form before checking each write and conversion.
func (c *subsetChecks) matchAssignment(statement *ast.AssignStmt) string {
	if len(statement.Lhs) != 1 || len(statement.Rhs) != 1 {
		if !c.pairAssignment(statement) {
			return "multiple assignments involving protected storage are unsupported"
		}
	}
	for _, destination := range statement.Lhs {
		if rule := c.matchWrite(destination); rule != "" {
			return rule
		}
	}
	source := statement.Rhs[0]
	if len(statement.Lhs) == 1 {
		if rule := c.matchBuilderBinding(statement.Lhs[0], source, statement.Tok == token.DEFINE); rule != "" {
			return rule
		}
	}
	for index, destination := range statement.Lhs {
		if rule := c.matchBinding(c.bindingSource(source, index), destination); rule != "" {
			return rule
		}
	}
	return ""
}

// matchBinding permits Local pointers only in local variable bindings.
func (c *subsetChecks) matchBinding(source types.Type, destination ast.Expr) string {
	if rule := c.matchTransport(source, c.pass.TypesInfo.TypeOf(destination)); rule != "" {
		return rule
	}
	if !localReference(source) && !c.queries.Locals(source) {
		return ""
	}
	name, ok := ast.Unparen(destination).(*ast.Ident)
	if !ok {
		if localReference(source) {
			return "Local pointers cannot enter aggregate storage"
		}
		return ""
	}
	if name.Name == "_" {
		return ""
	}
	variable, ok := c.pass.TypesInfo.ObjectOf(name).(*types.Var)
	if ok && variable.Parent() != c.pass.Pkg.Scope() {
		return ""
	}
	return "Local values and pointers cannot be stored in package variables"
}

// matchDeclaration accepts one binding or an exact two-result Get or Start call.
func (c *subsetChecks) matchDeclaration(declaration *ast.ValueSpec) string {
	if group, ok := c.parents[declaration].(*ast.GenDecl); ok && group.Tok != token.VAR {
		return "protected operation is outside the supported statement forms"
	}
	if len(declaration.Names) != 1 || len(declaration.Values) > 1 {
		if !c.pairDeclaration(declaration) {
			return "multiple assignments involving protected storage are unsupported"
		}
	}
	for _, name := range declaration.Names {
		variable := c.pass.TypesInfo.Defs[name]
		if variable != nil && name.Name != "_" && c.queries.Locals(variable.Type()) &&
			variable.Parent() == c.pass.Pkg.Scope() {
			return "Local values and pointers cannot be stored in package variables"
		}
		if variable != nil && localReference(variable.Type()) && variable.Parent() == c.pass.Pkg.Scope() {
			return "global Local pointers are unsupported"
		}
	}
	if len(declaration.Values) == 0 {
		return ""
	}
	source := declaration.Values[0]
	if len(declaration.Names) == 1 {
		if rule := c.matchBuilderBinding(declaration.Names[0], source, true); rule != "" {
			return rule
		}
	}
	for index, destination := range declaration.Names {
		if rule := c.matchBinding(c.bindingSource(source, index), destination); rule != "" {
			return rule
		}
	}
	return ""
}

// bindingSource resolves a whole value or one component of a compiler-checked tuple.
func (c *subsetChecks) bindingSource(source ast.Expr, index int) types.Type {
	t := c.pass.TypesInfo.TypeOf(source)
	if tuple, ok := t.(*types.Tuple); ok {
		return tuple.At(index).Type()
	}
	return t
}

// matchReturn maps each explicit result to its declared result type.
func (c *subsetChecks) matchReturn(statement *ast.ReturnStmt) string {
	for index, expression := range statement.Results {
		source := c.pass.TypesInfo.TypeOf(expression)
		if tuple, ok := source.(*types.Tuple); ok {
			for item := 0; item < tuple.Len(); item++ {
				if rule := c.matchReturnValue(tuple.At(item).Type(), c.returnType(statement, item)); rule != "" {
					return rule
				}
			}
		} else if rule := c.matchReturnValue(source, c.returnType(statement, index)); rule != "" {
			return rule
		}
	}
	return ""
}

// matchReturnValue permits concrete values and Frozen pointers at a return boundary.
func (c *subsetChecks) matchReturnValue(source, destination types.Type) string {
	if localReference(source) || localReference(destination) {
		return "returning Local pointers is unsupported"
	}
	return c.matchTransport(source, destination)
}

// matchSwitch checks every case conversion and excludes undocumented pointer switch operands.
func (c *subsetChecks) matchSwitch(statement *ast.SwitchStmt) string {
	if statement.Tag == nil {
		return ""
	}
	destination := c.pass.TypesInfo.TypeOf(statement.Tag)
	if c.protectedReference(destination) {
		return "reference operation is outside the supported allowlist"
	}
	for _, node := range statement.Body.List {
		for _, value := range node.(*ast.CaseClause).List {
			source := c.pass.TypesInfo.TypeOf(value)
			if rule := c.matchTransport(source, destination); rule != "" {
				return rule
			}
			if c.protectedReference(source) {
				return "reference operation is outside the supported allowlist"
			}
		}
	}
	return ""
}

// matchRange matches zero targets, fresh targets, or one existing target, then resolves yielded types.
func (c *subsetChecks) matchRange(statement *ast.RangeStmt) string {
	if statement.Tok == token.ASSIGN && statement.Key != nil && statement.Value != nil {
		return "multiple assignments involving protected storage are unsupported"
	}
	source := c.pass.TypesInfo.TypeOf(statement.X)
	if c.queries.Callable(source) {
		call, ok := ast.Unparen(statement.X).(*ast.CallExpr)
		if !ok {
			return "protected iteration requires a direct frozen All call"
		}
		selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "All" || !c.exactCollectionMethod(selector) {
			return "protected iteration requires a direct frozen All call"
		}
	}
	yielded := rangeTypes(source)
	for index, target := range []ast.Expr{statement.Key, statement.Value} {
		if target == nil {
			continue
		}
		if statement.Tok == token.ASSIGN {
			if rule := c.matchWrite(target); rule != "" {
				return rule
			}
		}
		if rule := c.matchBinding(yielded[index], target); rule != "" {
			return rule
		}
	}
	return ""
}

// rangeTypes reads the compiler-checked native or iterator element types.
func rangeTypes(t types.Type) [2]types.Type {
	switch source := t.Underlying().(type) {
	case *types.Array:
		return [2]types.Type{types.Typ[types.Int], source.Elem()}
	case *types.Slice:
		return [2]types.Type{types.Typ[types.Int], source.Elem()}
	case *types.Map:
		return [2]types.Type{source.Key(), source.Elem()}
	case *types.Signature:
		yield := source.Params().At(0).Type().Underlying().(*types.Signature)
		var result [2]types.Type
		for i := 0; i < yield.Params().Len(); i++ {
			result[i] = yield.Params().At(i).Type()
		}
		return result
	case *types.Basic:
		if source.Info()&types.IsString != 0 {
			return [2]types.Type{types.Typ[types.Int], types.Typ[types.Rune]}
		}
		return [2]types.Type{t, nil}
	}
	return [2]types.Type{}
}

// matchClosure identifies captures by declaration scope, including forward and imported globals.
func (c *subsetChecks) matchClosure(literal *ast.FuncLit) {
	functionScope := c.pass.TypesInfo.Scopes[literal.Type]
	ast.Inspect(literal.Body, func(node ast.Node) bool {
		name, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		variable, ok := c.pass.TypesInfo.Uses[name].(*types.Var)
		if !ok || variable.IsField() || !c.queries.Contains(variable.Type()) {
			return true
		}
		for scope := variable.Parent(); scope != nil; scope = scope.Parent() {
			if scope == functionScope {
				return true
			}
		}
		c.report(name, "closures cannot capture immutable storage")
		return true
	})
}

// protectedResult inspects multi-result expressions without ignoring a protected tuple component.
func (c *subsetChecks) protectedResult(t types.Type) bool {
	if c.queries.Contains(t) {
		return true
	}
	tuple, ok := t.(*types.Tuple)
	if !ok {
		return false
	}
	for i := 0; i < tuple.Len(); i++ {
		if c.queries.Contains(tuple.At(i).Type()) {
			return true
		}
	}
	return false
}

// exactCollectionMethod recognizes only the fixed library receiver rather than promoted wrapper methods.
func (c *subsetChecks) exactCollectionMethod(selector *ast.SelectorExpr) bool {
	selection := c.pass.TypesInfo.Selections[selector]
	if selection == nil || selection.Kind() != types.MethodVal || !libraryObject(selection.Obj()) {
		return false
	}
	receiver := selection.Recv()
	if pointer, ok := receiver.Underlying().(*types.Pointer); ok {
		receiver = pointer.Elem()
	}
	if !checker.IsCollection(receiver) && !checker.IsProjection(receiver) {
		return false
	}
	switch selection.Obj().Name() {
	case "Get", "At", "AtPointer", "Len", "All", "Clone", "Equal", "EqualString", "Unsafe", "Read", "Projection":
		return true
	}
	return false
}

// getCall permits the native two-result map read without admitting arbitrary multiple assignments.
func (c *subsetChecks) getCall(values []ast.Expr) bool {
	if len(values) != 1 {
		return false
	}
	call, ok := ast.Unparen(values[0]).(*ast.CallExpr)
	if !ok {
		return false
	}
	selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	return ok && selector.Sel.Name == "Get" && c.exactCollectionMethod(selector)
}

// pairCall accepts only the two-result functions listed in the source contract.
func (c *subsetChecks) pairCall(values []ast.Expr) bool {
	if c.getCall(values) {
		return true
	}
	if len(values) != 1 {
		return false
	}
	call, ok := ast.Unparen(values[0]).(*ast.CallExpr)
	if !ok {
		return false
	}
	object := c.calledObject(call.Fun)
	return runtimeObject(object) && object.Name() == "Start"
}

// pairAssignment requires new locals for Get and permits reused locals for Start.
func (c *subsetChecks) pairAssignment(statement *ast.AssignStmt) bool {
	if len(statement.Lhs) != 2 || !c.pairCall(statement.Rhs) {
		return false
	}
	fresh := c.getCall(statement.Rhs)
	if statement.Tok != token.DEFINE && (fresh || statement.Tok != token.ASSIGN) {
		return false
	}
	for _, left := range statement.Lhs {
		name, ok := ast.Unparen(left).(*ast.Ident)
		if !ok {
			return false
		}
		if name.Name == "_" {
			continue
		}
		variable, ok := c.pass.TypesInfo.ObjectOf(name).(*types.Var)
		if !ok || variable.Parent() == c.pass.Pkg.Scope() || fresh && c.pass.TypesInfo.Defs[name] == nil {
			return false
		}
	}
	return true
}

// pairDeclaration accepts two new local variables for a direct Get or Start call.
func (c *subsetChecks) pairDeclaration(statement *ast.ValueSpec) bool {
	if len(statement.Names) != 2 || !c.pairCall(statement.Values) {
		return false
	}
	_, local := c.parents[c.parents[statement]].(*ast.DeclStmt)
	return local
}
