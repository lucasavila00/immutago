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
		if statement.Recv != nil && len(statement.Recv.List) > 0 &&
			checker.Contains(c.pass.TypesInfo.TypeOf(statement.Recv.List[0].Type)) {
			c.report(statement.Name, "user-defined methods on protected receivers are unsupported")
		}
	case *ast.AssignStmt:
		c.fixedAssignment(statement)
	case *ast.ValueSpec:
		c.fixedDeclaration(statement)
	case *ast.RangeStmt:
		c.rangeBoundary(statement)
	case *ast.TypeSwitchStmt:
		c.typeSwitch(statement)
	case *ast.CommClause:
		c.selectCommunication(statement)
	}
}

// ownerAddress identifies the one supported local reference initializer; address checks validate its owner.
func ownerAddress(expression ast.Expr) bool {
	address, ok := ast.Unparen(expression).(*ast.UnaryExpr)
	return ok && address.Op == token.AND
}

// fixedAssignment permits only one-time local reference initialization, and single protected writes.
func (c *subsetChecks) fixedAssignment(statement *ast.AssignStmt) {
	if len(statement.Lhs) > 1 && c.protectedAssignment(statement) && !c.freshGetAssignment(statement) {
		c.report(statement, "multiple assignments involving protected storage are unsupported")
	}
	for _, destination := range statement.Lhs {
		name, ok := ast.Unparen(destination).(*ast.Ident)
		if !ok || name.Name == "_" || !protectedReference(c.pass.TypesInfo.TypeOf(name)) {
			continue
		}
		if statement.Tok == token.DEFINE && c.pass.TypesInfo.Defs[name] != nil &&
			len(statement.Lhs) == 1 && len(statement.Rhs) == 1 && ownerAddress(statement.Rhs[0]) {
			continue
		}
		c.report(name, "references require one fixed declaration from an owned local")
	}
}

// fixedDeclaration rejects uninitialized references, copies, nil initializers, and multiple protected declarations.
func (c *subsetChecks) fixedDeclaration(statement *ast.ValueSpec) {
	protected := c.protectedDeclarationValues(statement.Values)
	for _, name := range statement.Names {
		object := c.pass.TypesInfo.Defs[name]
		if object == nil {
			continue
		}
		protected = protected || checker.Contains(object.Type())
		if !protectedReference(object.Type()) {
			continue
		}
		if len(statement.Names) == 1 && len(statement.Values) == 1 && ownerAddress(statement.Values[0]) {
			continue
		}
		c.report(name, "references require one fixed declaration from an owned local")
	}
	if protected && len(statement.Names) > 1 && !c.getDeclaration(statement) {
		c.report(statement, "multiple assignments involving protected storage are unsupported")
	}
}

// protectedAssignment includes scalar fields of owners and tuple-valued protected results.
func (c *subsetChecks) protectedAssignment(statement *ast.AssignStmt) bool {
	for _, expression := range append(append([]ast.Expr(nil), statement.Lhs...), statement.Rhs...) {
		if c.protectedStorage(expression) || protectedResult(c.pass.TypesInfo.TypeOf(expression)) {
			return true
		}
	}
	return false
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
	if selection == nil || !libraryObject(selection.Obj()) {
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
	for _, target := range []ast.Expr{statement.Key, statement.Value} {
		if target != nil && protectedReference(c.pass.TypesInfo.TypeOf(target)) {
			c.report(target, "references require one fixed declaration from an owned local")
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
