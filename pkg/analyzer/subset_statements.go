package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// statementBoundary rejects source forms whose ordering or alias transfer is outside the subset.
func (c *subsetChecks) statementBoundary(node ast.Node) {
	switch statement := node.(type) {
	case *ast.FuncDecl:
		signature := c.pass.TypesInfo.Defs[statement.Name].Type().(*types.Signature)
		if genericSignature(signature) {
			c.genericStorage(statement, statement.Name)
		}
		if signature.Recv() != nil && (checker.Contains(signature.Recv().Type()) || protectedCallable(signature)) {
			c.report(statement.Name, "user-defined methods with immutable storage are unsupported")
		}
	case *ast.ValueSpec:
		c.declarationShape(statement)
	case *ast.TypeSpec:
		if statement.TypeParams != nil {
			c.genericStorage(statement, statement.Name)
		}
	case *ast.RangeStmt:
		c.rangeBoundary(statement)
	case *ast.TypeSwitchStmt:
		c.typeSwitch(statement)
	case *ast.CommClause:
		c.selectCommunication(statement)
	}
}

// declarationShape requires one protected binding, except for a direct frozen map lookup.
func (c *subsetChecks) declarationShape(statement *ast.ValueSpec) {
	protected := false
	for _, name := range statement.Names {
		object := c.pass.TypesInfo.Defs[name]
		if object == nil {
			continue
		}
		protected = protected || checker.Contains(object.Type())
	}
	if protected && len(statement.Names) > 1 && !c.getDeclaration(statement) {
		c.report(statement, "multiple assignments involving protected storage are unsupported")
	}
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

// selectCommunication inspects only the channel operation, leaving case bodies to ordinary checking.
func (c *subsetChecks) selectCommunication(clause *ast.CommClause) {
	if clause.Comm == nil {
		return
	}
	ast.Inspect(clause.Comm, func(node ast.Node) bool {
		expression, ok := node.(ast.Expr)
		if ok && (c.protectedRoot(expression) || c.ownedExpression(expression)) {
			c.report(expression, "protected storage is unsupported in select communication clauses")
		}
		return true
	})
}

// typeSwitch rejects recovering protected storage from an interface through case types.
func (c *subsetChecks) typeSwitch(statement *ast.TypeSwitchStmt) {
	for _, node := range statement.Body.List {
		clause := node.(*ast.CaseClause)
		for _, expression := range clause.List {
			if checker.Contains(c.pass.TypesInfo.TypeOf(expression)) {
				c.report(expression, "protected storage is unsupported in type switches")
			}
		}
	}
}

// synchronousStorage keeps protected values and native builder handles out of go and defer calls.
func (c *subsetChecks) synchronousStorage(expression ast.Expr) {
	if c.pass.TypesInfo.Types[expression].IsType() ||
		(!c.protectedRoot(expression) && !c.ownedExpression(expression)) {
		return
	}
	for parent := c.parents[expression]; parent != nil; parent = c.parents[parent] {
		switch parent.(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			return
		case *ast.GoStmt, *ast.DeferStmt:
			c.report(expression, "protected storage requires synchronous calls")
			return
		}
	}
}

// rangeBoundary rejects implicit reference creation and unknown protected callback iteration.
func (c *subsetChecks) rangeBoundary(statement *ast.RangeStmt) {
	c.rangeTransport(statement)
	for _, target := range []ast.Expr{statement.Key, statement.Value} {
		if target != nil && statement.Tok == token.ASSIGN {
			c.write(target)
		}
	}
	if !protectedCallable(c.pass.TypesInfo.TypeOf(statement.X)) {
		return
	}
	call, ok := ast.Unparen(statement.X).(*ast.CallExpr)
	if ok {
		selector, selected := ast.Unparen(call.Fun).(*ast.SelectorExpr)
		if selected && selector.Sel.Name == "All" && c.exactCollectionMethod(selector) {
			return
		}
	}
	c.report(statement.X, "protected iteration requires a direct frozen All call")
}

// rangeTransport checks implicit assignments from native storage and direct frozen iterators.
func (c *subsetChecks) rangeTransport(statement *ast.RangeStmt) {
	t := c.pass.TypesInfo.TypeOf(statement.X)
	if !checker.Contains(t) && !protectedCallable(t) {
		return
	}
	var key, value types.Type
	switch source := t.Underlying().(type) {
	case *types.Array:
		key, value = types.Typ[types.Int], source.Elem()
	case *types.Slice:
		key, value = types.Typ[types.Int], source.Elem()
	case *types.Map:
		key, value = source.Key(), source.Elem()
	case *types.Signature:
		// The Go compiler has checked the range function's yield signature.
		yield := source.Params().At(0).Type().Underlying().(*types.Signature)
		if yield.Params().Len() > 0 {
			key = yield.Params().At(0).Type()
		}
		if yield.Params().Len() > 1 {
			value = yield.Params().At(1).Type()
		}
	}
	if statement.Key != nil {
		c.transportType(statement.Key, key, c.pass.TypesInfo.TypeOf(statement.Key))
	}
	if statement.Value != nil {
		c.transportType(statement.Value, value, c.pass.TypesInfo.TypeOf(statement.Value))
	}
}
