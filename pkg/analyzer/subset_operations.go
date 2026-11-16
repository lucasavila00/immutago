package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// localReference identifies a pointer to a concrete im.Local struct.
func localReference(t types.Type) bool {
	if t == nil {
		return false
	}
	pointer, ok := t.Underlying().(*types.Pointer)
	return ok && checker.Local(pointer.Elem())
}

// operationExpression separates runtime operands from compiler type and field metadata.
func (c *subsetChecks) operationExpression(expression ast.Expr) bool {
	if selector, ok := c.parents[expression].(*ast.SelectorExpr); ok && selector.Sel == expression {
		return false
	}
	if c.pass.TypesInfo.Types[expression].IsType() {
		return false
	}
	if name, ok := expression.(*ast.Ident); ok {
		object := c.pass.TypesInfo.ObjectOf(name)
		if c.pass.TypesInfo.Defs[name] != nil {
			return false
		}
		if _, typeName := object.(*types.TypeName); typeName {
			return false
		}
		if field, ok := object.(*types.Var); ok && field.IsField() {
			return false
		}
	}
	if object := c.calledObject(expression); libraryObject(object) {
		if _, function := object.(*types.Func); function {
			return true
		}
	}
	return c.protectedStorage(expression) || c.queries.Contains(c.pass.TypesInfo.TypeOf(expression)) ||
		c.queries.Callable(c.pass.TypesInfo.TypeOf(expression))
}

// matchOperation requires a known producer, consuming operation, and complete statement.
func (c *subsetChecks) matchOperation(expression ast.Expr) (ast.Node, checker.Failure) {
	if rule := c.matchSource(expression); rule.Reason != "" {
		if call := c.directCall(expression); call != nil {
			return call, rule
		}
		return expression, rule
	}
	if rule := c.matchUse(expression); rule.Reason != "" {
		operation := c.parents[expression]
		if _, pair := operation.(*ast.KeyValueExpr); pair {
			operation = c.parents[operation]
		}
		return operation, rule
	}
	return c.statementOperation(expression), c.matchStatement(expression)
}

// matchSource lists the complete producers of protected values and pointers.
func (c *subsetChecks) matchSource(expression ast.Expr) checker.Failure {
	if pointer, ok := c.pass.TypesInfo.TypeOf(expression).Underlying().(*types.Pointer); ok &&
		c.protectedReference(pointer) && !checker.Marked(pointer.Elem()) {
		return shapeFailure("immutable references require a concrete marked struct target")
	}
	switch source := expression.(type) {
	case *ast.Ident:
		return c.matchIdentifier(source)
	case *ast.ParenExpr:
		return checker.Failure{}
	case *ast.SelectorExpr:
		return c.matchSelector(source)
	case *ast.IndexExpr:
		return c.matchIndex(source)
	case *ast.IndexListExpr:
		return c.matchFunctionValue(source)
	case *ast.CompositeLit:
		return c.matchLiteral(source)
	case *ast.CallExpr:
		return c.matchCall(source)
	case *ast.StarExpr:
		if c.readablePointer(source.X) {
			return checker.Failure{}
		}
	case *ast.UnaryExpr:
		return c.matchUnary(source)
	case *ast.SliceExpr:
		if stringType(c.pass.TypesInfo.TypeOf(source.X)) || !c.protectedStorage(source.X) {
			return checker.Failure{}
		}
		return shapeFailure("slice projections of immutable storage are unsupported")
	case *ast.TypeAssertExpr:
		return c.matchAssertion(source)
	case *ast.FuncLit:
		if !c.queries.Callable(c.pass.TypesInfo.TypeOf(source)) {
			return checker.Failure{}
		}
		return shapeFailure("callbacks require ordinary or persistent Frozen parameters and results")
	}
	return shapeFailure("protected operation is outside the supported expression forms")
}

// matchAssertion checks known Frozen contracts before recovering a persistent concrete value.
func (c *subsetChecks) matchAssertion(expression *ast.TypeAssertExpr) checker.Failure {
	destination := c.pass.TypesInfo.TypeOf(expression.Type)
	if checker.FrozenInterface(destination) {
		source := c.pass.TypesInfo.TypeOf(expression.X)
		failure := c.matchFrozenInterface(source, destination)
		if failure.Reason != "" && !checker.FrozenInterface(source) && c.proofs.FrozenReason(destination) == "" {
			failure.Repair = "assert the concrete Frozen record value or pointer first, then convert it " +
				"to the Frozen interface; for example, tree := boxed.(*Tree); scorer := Scorer(tree)"
		}
		return failure
	}
	if c.queries.Transport(destination) {
		return shapeFailure("interface assertions require ordinary or persistent Frozen values")
	}
	return checker.Failure{}
}

// matchIdentifier separates variable reads from direct function and builtin callees.
func (c *subsetChecks) matchIdentifier(name *ast.Ident) checker.Failure {
	switch c.pass.TypesInfo.ObjectOf(name).(type) {
	case *types.Var, *types.Nil:
		if !c.queries.Callable(c.pass.TypesInfo.TypeOf(name)) {
			return checker.Failure{}
		}
		return c.matchFunctionValue(name)
	case *types.Func, *types.Builtin:
		return c.matchFunctionValue(name)
	}
	return shapeFailure("protected operation is outside the supported expression forms")
}

// matchUse resolves the complete operation that consumes a protected operand.
func (c *subsetChecks) matchUse(expression ast.Expr) checker.Failure {
	if _, statement := c.parents[expression].(ast.Stmt); statement {
		return c.matchStatement(expression)
	}
	switch parent := c.parents[expression].(type) {
	case *ast.ParenExpr:
		return checker.Failure{}
	case *ast.SelectorExpr:
		return c.matchSelector(parent)
	case *ast.IndexExpr:
		return c.matchIndex(parent)
	case *ast.IndexListExpr:
		return c.matchFunctionValue(parent)
	case *ast.StarExpr:
		if parent.X == expression && c.readablePointer(expression) {
			return checker.Failure{}
		}
	case *ast.UnaryExpr:
		return c.matchUnary(parent)
	case *ast.BinaryExpr:
		return c.matchBinary(parent)
	case *ast.SliceExpr:
		if stringType(c.pass.TypesInfo.TypeOf(parent.X)) || !c.protectedStorage(parent.X) {
			return checker.Failure{}
		}
		return shapeFailure("slice projections of immutable storage are unsupported")
	case *ast.CallExpr:
		return c.matchCall(parent)
	case *ast.CompositeLit:
		return c.matchLiteral(parent)
	case *ast.KeyValueExpr:
		if literal, ok := c.parents[parent].(*ast.CompositeLit); ok {
			return c.matchLiteral(literal)
		}
	case *ast.ValueSpec:
		return c.matchDeclaration(parent)
	}
	if !c.restrictedOperand(expression) {
		return checker.Failure{}
	}
	return shapeFailure("protected operation is outside the supported expression forms")
}

// readablePointer permits checked protected pointers and ordinary holder pointers.
func (c *subsetChecks) readablePointer(expression ast.Expr) bool {
	return c.protectedReference(c.pass.TypesInfo.TypeOf(expression)) || !c.protectedRoot(expression)
}

// matchFunctionValue permits protected functions only as direct callees.
func (c *subsetChecks) matchFunctionValue(expression ast.Expr) checker.Failure {
	if call := c.directCall(expression); call != nil {
		return c.matchCall(call)
	}
	object := c.calledObject(expression)
	if libraryObject(object) && (object.Name() == "FreezeMap" || object.Name() == "FreezeSlice") {
		return shapeFailure("freeze functions must be called directly")
	}
	if !c.queries.Callable(c.pass.TypesInfo.TypeOf(expression)) {
		return checker.Failure{}
	}
	return shapeFailure("callbacks require ordinary or persistent Frozen parameters and results")
}

// matchSelector distinguishes fields, package variables, and the supported method calls.
func (c *subsetChecks) matchSelector(selector *ast.SelectorExpr) checker.Failure {
	selection := c.pass.TypesInfo.Selections[selector]
	if selection == nil {
		if _, variable := c.pass.TypesInfo.ObjectOf(selector.Sel).(*types.Var); variable {
			return checker.Failure{}
		}
		return c.matchFunctionValue(selector)
	}
	if selection.Kind() == types.FieldVal {
		if c.queries.Callable(c.pass.TypesInfo.TypeOf(selector)) {
			return c.matchFunctionValue(selector)
		}
		return checker.Failure{}
	}
	if selection.Kind() == types.MethodExpr && libraryObject(selection.Obj()) &&
		c.queries.Callable(c.pass.TypesInfo.TypeOf(selector)) {
		return shapeFailure("method expressions require ordinary or persistent Frozen receivers and contracts")
	}
	if c.ownedSlotSelector(selector) && c.directCall(selector) != nil {
		return checker.Failure{}
	}
	if c.exactCollectionMethod(selector) {
		if c.directCall(selector) == nil && (c.queries.Transport(selection.Recv()) ||
			c.queries.Callable(c.pass.TypesInfo.TypeOf(selector))) {
			return shapeFailure("method values require ordinary or persistent Frozen receivers and contracts")
		}
		return checker.Failure{}
	}
	return c.matchMethod(selector)
}

// matchUnary permits fresh marked allocations and ordinary scalar operations.
func (c *subsetChecks) matchUnary(expression *ast.UnaryExpr) checker.Failure {
	if expression.Op == token.AND {
		return c.matchAddress(expression)
	}
	switch expression.Op {
	case token.ARROW:
		if !c.protectedStorage(expression.X) {
			return checker.Failure{}
		}
	case token.ADD, token.SUB, token.NOT, token.XOR:
		if _, scalar := c.pass.TypesInfo.TypeOf(expression.X).Underlying().(*types.Basic); scalar {
			return checker.Failure{}
		}
	}
	return shapeFailure("protected operation is outside the supported expression forms")
}

// matchAddress separates ordinary slots, generated field addresses, and fresh protected roots.
func (c *subsetChecks) matchAddress(expression *ast.UnaryExpr) checker.Failure {
	if !c.protectedStorage(expression.X) {
		return checker.Failure{}
	}
	pointer := c.pass.TypesInfo.TypeOf(expression).Underlying().(*types.Pointer)
	if c.generatedSource(expression) && checker.Frozen(pointer.Elem()) {
		return checker.Failure{}
	}
	if !c.queries.Owns(c.pass.TypesInfo.TypeOf(expression.X)) && !checker.Marked(c.pass.TypesInfo.TypeOf(expression.X)) {
		return shapeFailure("interior addresses of protected storage are unsupported")
	}
	if _, fresh := ast.Unparen(expression.X).(*ast.CompositeLit); !fresh {
		return shapeFailure("immutable references require fresh allocation")
	}
	if checker.Marked(pointer.Elem()) {
		return checker.Failure{}
	}
	return shapeFailure("immutable references require a concrete marked struct target")
}

// matchBinary checks implicit comparisons before permitting compiler-checked scalar operators.
func (c *subsetChecks) matchBinary(expression *ast.BinaryExpr) checker.Failure {
	left, right := c.pass.TypesInfo.TypeOf(expression.X), c.pass.TypesInfo.TypeOf(expression.Y)
	if rule := c.matchTransport(left, right); rule.Reason != "" {
		return rule
	}
	if rule := c.matchTransport(right, left); rule.Reason != "" {
		return rule
	}
	if c.queries.Owns(left) || c.queries.Owns(right) {
		if expression.Op != token.EQL && expression.Op != token.NEQ {
			return shapeFailure("protected operation is outside the supported expression forms")
		}
	}
	return checker.Failure{}
}

// matchIndex permits value-array reads and native builder reads with concrete map keys.
func (c *subsetChecks) matchIndex(expression *ast.IndexExpr) checker.Failure {
	if _, callable := c.pass.TypesInfo.TypeOf(expression).Underlying().(*types.Signature); callable {
		return c.matchFunctionValue(expression)
	}
	switch source := c.pass.TypesInfo.TypeOf(expression.X).Underlying().(type) {
	case *types.Array:
		return checker.Failure{}
	case *types.Map:
		return c.matchTransport(c.pass.TypesInfo.TypeOf(expression.Index), source.Key())
	case *types.Slice, *types.Basic:
		return checker.Failure{}
	case *types.Pointer:
		if _, array := source.Elem().Underlying().(*types.Array); array && !c.queries.Owns(source) {
			return checker.Failure{}
		}
	}
	return shapeFailure("reference operation is outside the supported allowlist")
}
