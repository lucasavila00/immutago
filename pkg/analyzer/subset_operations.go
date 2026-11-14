package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// localReference identifies a pointer to a concrete im.LocalRef struct.
func localReference(t types.Type) bool {
	if t == nil {
		return false
	}
	pointer, ok := t.Underlying().(*types.Pointer)
	return ok && checker.LocalRef(pointer.Elem())
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
	return c.protectedStorage(expression) || protectedResult(c.pass.TypesInfo.TypeOf(expression)) ||
		protectedCallable(c.pass.TypesInfo.TypeOf(expression))
}

// matchOperation requires a known producer, consuming operation, and complete statement.
func (c *subsetChecks) matchOperation(expression ast.Expr) string {
	if rule := c.matchSource(expression); rule != "" {
		return rule
	}
	if rule := c.matchUse(expression); rule != "" {
		return rule
	}
	return c.matchStatement(expression)
}

// matchSource lists the complete producers of protected values and pointers.
func (c *subsetChecks) matchSource(expression ast.Expr) string {
	if pointer, ok := c.pass.TypesInfo.TypeOf(expression).Underlying().(*types.Pointer); ok &&
		protectedReference(pointer) && !checker.Marked(pointer.Elem()) {
		return "immutable references require a concrete marked struct target"
	}
	switch source := expression.(type) {
	case *ast.Ident:
		switch c.pass.TypesInfo.ObjectOf(source).(type) {
		case *types.Var, *types.Nil:
			if protectedCallable(c.pass.TypesInfo.TypeOf(source)) {
				return c.matchFunctionValue(source)
			}
			return ""
		case *types.Func, *types.Builtin:
			return c.matchFunctionValue(source)
		}
	case *ast.ParenExpr:
		return ""
	case *ast.SelectorExpr:
		return c.matchSelector(source)
	case *ast.IndexExpr:
		if _, callable := c.pass.TypesInfo.TypeOf(source).Underlying().(*types.Signature); callable {
			return c.matchFunctionValue(source)
		}
		return c.matchIndex(source)
	case *ast.IndexListExpr:
		return c.matchFunctionValue(source)
	case *ast.CompositeLit:
		return c.matchLiteral(source)
	case *ast.CallExpr:
		return c.matchCall(source)
	case *ast.StarExpr:
		if protectedReference(c.pass.TypesInfo.TypeOf(source.X)) {
			return ""
		}
	case *ast.UnaryExpr:
		return c.matchUnary(source)
	case *ast.SliceExpr:
		if stringType(c.pass.TypesInfo.TypeOf(source.X)) {
			return ""
		}
		return "slice views of immutable storage are unsupported"
	case *ast.TypeAssertExpr:
		return "immutable interface assertions are unsupported"
	case *ast.FuncLit:
		return "immutable callbacks are unsupported"
	}
	return "protected operation is outside the supported expression forms"
}

// matchUse resolves the complete operation that consumes a protected operand.
func (c *subsetChecks) matchUse(expression ast.Expr) string {
	if _, statement := c.parents[expression].(ast.Stmt); statement {
		return c.matchStatement(expression)
	}
	switch parent := c.parents[expression].(type) {
	case *ast.ParenExpr:
		return ""
	case *ast.SelectorExpr:
		return c.matchSelector(parent)
	case *ast.IndexExpr:
		if _, callable := c.pass.TypesInfo.TypeOf(parent).Underlying().(*types.Signature); callable {
			return c.matchFunctionValue(parent)
		}
		return c.matchIndex(parent)
	case *ast.IndexListExpr:
		return c.matchFunctionValue(parent)
	case *ast.StarExpr:
		if parent.X == expression && protectedReference(c.pass.TypesInfo.TypeOf(expression)) {
			return ""
		}
	case *ast.UnaryExpr:
		return c.matchUnary(parent)
	case *ast.BinaryExpr:
		return c.matchBinary(parent)
	case *ast.SliceExpr:
		if stringType(c.pass.TypesInfo.TypeOf(parent.X)) {
			return ""
		}
		return "slice views of immutable storage are unsupported"
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
	return "protected operation is outside the supported expression forms"
}

// matchFunctionValue permits protected functions only as direct callees.
func (c *subsetChecks) matchFunctionValue(expression ast.Expr) string {
	if call := c.directCall(expression); call != nil {
		return c.matchCall(call)
	}
	object := c.calledObject(expression)
	if libraryObject(object) && (object.Name() == "FreezeMap" || object.Name() == "FreezeSlice") {
		return "freeze functions must be called directly"
	}
	return "immutable callbacks are unsupported"
}

// matchSelector distinguishes fields, package variables, and the fixed collection calls.
func (c *subsetChecks) matchSelector(selector *ast.SelectorExpr) string {
	selection := c.pass.TypesInfo.Selections[selector]
	if selection == nil {
		if _, variable := c.pass.TypesInfo.ObjectOf(selector.Sel).(*types.Var); variable {
			return ""
		}
		return c.matchFunctionValue(selector)
	}
	if selection.Kind() == types.FieldVal {
		if protectedCallable(c.pass.TypesInfo.TypeOf(selector)) {
			return c.matchFunctionValue(selector)
		}
		return ""
	}
	if selection.Kind() == types.MethodExpr && libraryObject(selection.Obj()) {
		return "immutable method expressions are unsupported"
	}
	if !c.exactCollectionMethod(selector) {
		return "user-defined methods with immutable storage are unsupported"
	}
	if c.directCall(selector) == nil {
		return "bound immutable methods are unsupported"
	}
	return ""
}

// matchUnary permits fresh marked allocations and ordinary scalar operations.
func (c *subsetChecks) matchUnary(expression *ast.UnaryExpr) string {
	if expression.Op == token.AND {
		if !checker.Contains(c.pass.TypesInfo.TypeOf(expression.X)) {
			return "scalar interior addresses are unsupported"
		}
		if _, fresh := ast.Unparen(expression.X).(*ast.CompositeLit); !fresh {
			return "immutable references require fresh allocation"
		}
		pointer, ok := c.pass.TypesInfo.TypeOf(expression).Underlying().(*types.Pointer)
		if ok && checker.Marked(pointer.Elem()) {
			return ""
		}
		return "immutable references require a concrete marked struct target"
	}
	switch expression.Op {
	case token.ADD, token.SUB, token.NOT, token.XOR:
		if _, scalar := c.pass.TypesInfo.TypeOf(expression.X).Underlying().(*types.Basic); scalar {
			return ""
		}
	}
	return "protected operation is outside the supported expression forms"
}

// matchBinary checks implicit comparisons before permitting compiler-checked scalar operators.
func (c *subsetChecks) matchBinary(expression *ast.BinaryExpr) string {
	left, right := c.pass.TypesInfo.TypeOf(expression.X), c.pass.TypesInfo.TypeOf(expression.Y)
	if rule := matchTransport(left, right); rule != "" {
		return rule
	}
	if rule := matchTransport(right, left); rule != "" {
		return rule
	}
	if checker.Contains(left) || checker.Contains(right) {
		if expression.Op != token.EQL && expression.Op != token.NEQ {
			return "protected operation is outside the supported expression forms"
		}
	}
	return ""
}

// matchIndex permits value-array reads and native builder reads with concrete map keys.
func (c *subsetChecks) matchIndex(expression *ast.IndexExpr) string {
	switch source := c.pass.TypesInfo.TypeOf(expression.X).Underlying().(type) {
	case *types.Array:
		return ""
	case *types.Map:
		return matchTransport(c.pass.TypesInfo.TypeOf(expression.Index), source.Key())
	case *types.Slice, *types.Basic:
		return ""
	}
	return "reference operation is outside the supported allowlist"
}
