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

// functionDeclaration checks the exact named-function contract.
func (c *subsetChecks) functionDeclaration(declaration *ast.FuncDecl) {
	signature := c.pass.TypesInfo.Defs[declaration.Name].Type().(*types.Signature)
	if rule := subsetResults(signature.Results()); rule != "" {
		c.report(declaration.Name, rule)
	}
	if genericSignature(signature) {
		c.genericStorage(declaration, declaration.Name)
	}
	if signature.Recv() != nil && (checker.Contains(signature.Recv().Type()) || protectedCallable(signature)) {
		c.report(declaration.Name, "user-defined methods with immutable storage are unsupported")
	}
}

// valueDeclaration activates binding checks even when the initializer is absent or nil.
func (c *subsetChecks) valueDeclaration(declaration *ast.ValueSpec) {
	for _, name := range declaration.Names {
		variable := c.pass.TypesInfo.Defs[name]
		if variable != nil && (checker.Contains(variable.Type()) || c.ownedExpression(name)) {
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
			if checker.Contains(c.pass.TypesInfo.TypeOf(expression)) {
				c.report(expression, "protected storage is unsupported in type switches")
			}
		}
	}
}

// matchAssignment resolves the whole binding form before checking each write and conversion.
func (c *subsetChecks) matchAssignment(statement *ast.AssignStmt) string {
	if len(statement.Lhs) != 1 || len(statement.Rhs) != 1 {
		if !c.freshGetAssignment(statement) {
			return "multiple assignments involving protected storage are unsupported"
		}
	}
	for _, destination := range statement.Lhs {
		if rule := c.matchWrite(destination); rule != "" {
			return rule
		}
	}
	for index, source := range statement.Rhs {
		if len(statement.Lhs) == 1 {
			if rule := c.matchBuilderBinding(statement.Lhs[0], source, statement.Tok == token.DEFINE); rule != "" {
				return rule
			}
		}
		if tuple, ok := c.pass.TypesInfo.TypeOf(source).(*types.Tuple); ok {
			for item := 0; item < tuple.Len(); item++ {
				if rule := c.matchBinding(tuple.At(item).Type(), statement.Lhs[item]); rule != "" {
					return rule
				}
			}
		} else if rule := c.matchBinding(c.pass.TypesInfo.TypeOf(source), statement.Lhs[index]); rule != "" {
			return rule
		}
	}
	return ""
}

// matchBinding permits LocalRef pointers only in local variable bindings.
func (c *subsetChecks) matchBinding(source types.Type, destination ast.Expr) string {
	if rule := matchTransport(source, c.pass.TypesInfo.TypeOf(destination)); rule != "" {
		return rule
	}
	if !localReference(source) {
		return ""
	}
	name, ok := ast.Unparen(destination).(*ast.Ident)
	if !ok {
		return "LocalRef pointers cannot enter aggregate storage"
	}
	if name.Name == "_" {
		return ""
	}
	variable, ok := c.pass.TypesInfo.ObjectOf(name).(*types.Var)
	if ok && variable.Parent() != c.pass.Pkg.Scope() {
		return ""
	}
	return "global LocalRef pointers are unsupported"
}

// matchDeclaration accepts one binding or the exact two-result frozen map reader.
func (c *subsetChecks) matchDeclaration(declaration *ast.ValueSpec) string {
	if len(declaration.Names) != 1 || len(declaration.Values) > 1 {
		if !c.getDeclaration(declaration) {
			return "multiple assignments involving protected storage are unsupported"
		}
	}
	for _, name := range declaration.Names {
		variable := c.pass.TypesInfo.Defs[name]
		if variable != nil && localReference(variable.Type()) && variable.Parent() == c.pass.Pkg.Scope() {
			return "global LocalRef pointers are unsupported"
		}
	}
	for index, source := range declaration.Values {
		if len(declaration.Names) == 1 {
			if rule := c.matchBuilderBinding(declaration.Names[0], source, true); rule != "" {
				return rule
			}
		}
		if tuple, ok := c.pass.TypesInfo.TypeOf(source).(*types.Tuple); ok {
			for item := 0; item < tuple.Len(); item++ {
				if rule := c.matchBinding(tuple.At(item).Type(), declaration.Names[item]); rule != "" {
					return rule
				}
			}
		} else if rule := c.matchBinding(c.pass.TypesInfo.TypeOf(source), declaration.Names[index]); rule != "" {
			return rule
		}
	}
	return ""
}

// matchReturn maps each explicit result to its declared result type.
func (c *subsetChecks) matchReturn(statement *ast.ReturnStmt) string {
	for index, expression := range statement.Results {
		source := c.pass.TypesInfo.TypeOf(expression)
		if tuple, ok := source.(*types.Tuple); ok {
			for item := 0; item < tuple.Len(); item++ {
				if rule := matchReturnValue(tuple.At(item).Type(), c.returnType(statement, item)); rule != "" {
					return rule
				}
			}
		} else if rule := matchReturnValue(source, c.returnType(statement, index)); rule != "" {
			return rule
		}
	}
	return ""
}

// matchReturnValue permits concrete values and Shared pointers at a return boundary.
func matchReturnValue(source, destination types.Type) string {
	if localReference(source) || localReference(destination) {
		return "returning LocalRef pointers is unsupported"
	}
	return matchTransport(source, destination)
}

// matchSwitch checks every case conversion and excludes undocumented pointer switch operands.
func (c *subsetChecks) matchSwitch(statement *ast.SwitchStmt) string {
	if statement.Tag == nil {
		return ""
	}
	destination := c.pass.TypesInfo.TypeOf(statement.Tag)
	if protectedReference(destination) {
		return "reference operation is outside the supported allowlist"
	}
	for _, node := range statement.Body.List {
		for _, value := range node.(*ast.CaseClause).List {
			source := c.pass.TypesInfo.TypeOf(value)
			if rule := matchTransport(source, destination); rule != "" {
				return rule
			}
			if protectedReference(source) {
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
	if protectedCallable(source) {
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
		if !ok || variable.IsField() || !checker.Contains(variable.Type()) {
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
func protectedResult(t types.Type) bool {
	if checker.Contains(t) {
		return true
	}
	tuple, ok := t.(*types.Tuple)
	if !ok {
		return false
	}
	for i := 0; i < tuple.Len(); i++ {
		if checker.Contains(tuple.At(i).Type()) {
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
	if !checker.IsCollection(receiver) {
		return false
	}
	switch selection.Obj().Name() {
	case "Get", "At", "Len", "All", "Clone":
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

// freshGetAssignment accepts map results only into newly declared local names.
func (c *subsetChecks) freshGetAssignment(statement *ast.AssignStmt) bool {
	if statement.Tok != token.DEFINE || len(statement.Lhs) != 2 || !c.getCall(statement.Rhs) {
		return false
	}
	for _, left := range statement.Lhs {
		name, ok := left.(*ast.Ident)
		if !ok || (name.Name != "_" && c.pass.TypesInfo.Defs[name] == nil) {
			return false
		}
	}
	return true
}

// getDeclaration accepts two new local variables for a direct map lookup.
func (c *subsetChecks) getDeclaration(statement *ast.ValueSpec) bool {
	if len(statement.Names) != 2 || !c.getCall(statement.Values) {
		return false
	}
	for _, name := range statement.Names {
		object := c.pass.TypesInfo.Defs[name]
		if object != nil && object.Parent() == c.pass.Pkg.Scope() {
			return false
		}
	}
	return true
}
