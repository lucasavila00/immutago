package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
	"golang.org/x/tools/go/analysis"
)

// ownedSlotSelector resolves a method on the sealed slot type.
func (c *subsetChecks) ownedSlotSelector(selector *ast.SelectorExpr) bool {
	selection := c.pass.TypesInfo.Selections[selector]
	if selection == nil || selection.Kind() != types.MethodVal || !libraryObject(selection.Obj()) {
		return false
	}
	receiver := selection.Recv()
	if pointer, ok := receiver.Underlying().(*types.Pointer); ok {
		receiver = pointer.Elem()
	}
	return checker.IsLibraryType(receiver, "Owned")
}

// ownedSlotCall recognizes a direct call on the sealed slot type.
func (c *subsetChecks) ownedSlotCall(call *ast.CallExpr) bool {
	selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	return ok && c.ownedSlotSelector(selector)
}

// containsSlotValue finds slots in value storage, without following pointers.
func containsSlotValue(t types.Type, seen map[types.Type]bool) bool {
	if t == nil || seen[t] {
		return false
	}
	seen[t] = true
	if checker.IsLibraryType(t, "Owned") {
		return true
	}
	switch typ := t.Underlying().(type) {
	case *types.Array:
		return containsSlotValue(typ.Elem(), seen)
	case *types.Slice:
		return containsSlotValue(typ.Elem(), seen)
	case *types.Map:
		return containsSlotValue(typ.Key(), seen) || containsSlotValue(typ.Elem(), seen)
	case *types.Chan:
		return containsSlotValue(typ.Elem(), seen)
	case *types.Signature:
		return containsSlotValue(typ.Params(), seen) || containsSlotValue(typ.Results(), seen)
	case *types.Tuple:
		return tupleHasSlotValue(typ, seen)
	case *types.Struct:
		return structHasSlotValue(typ, seen)
	}
	return false
}

// tupleHasSlotValue checks each parameter or result type.
func tupleHasSlotValue(tuple *types.Tuple, seen map[types.Type]bool) bool {
	for i := 0; i < tuple.Len(); i++ {
		if containsSlotValue(tuple.At(i).Type(), seen) {
			return true
		}
	}
	return false
}

// structHasSlotValue checks each field stored by value.
func structHasSlotValue(structure *types.Struct, seen map[types.Type]bool) bool {
	for field := range structure.Fields() {
		if containsSlotValue(field.Type(), seen) {
			return true
		}
	}
	return false
}

// hasSlotValue reports whether copying a value would copy an atomic slot.
func (c *subsetChecks) hasSlotValue(t types.Type) bool {
	if c.slots == nil {
		c.slots = make(map[types.Type]bool)
	}
	if result, ok := c.slots[t]; ok {
		return result
	}
	result := containsSlotValue(t, make(map[types.Type]bool))
	c.slots[t] = result
	return result
}

// slotRelevant follows pointers to find slot-bearing types at package boundaries.
func (c *subsetChecks) slotRelevant(t types.Type) bool {
	seen := make(map[types.Type]bool)
	for t != nil && !seen[t] {
		seen[t] = true
		if c.hasSlotValue(t) {
			return true
		}
		pointer, ok := t.Underlying().(*types.Pointer)
		if !ok {
			return false
		}
		t = pointer.Elem()
	}
	return false
}

// checkOwned rejects all slot and holder value operations except the listed forms.
func checkOwned(pass *analysis.Pass) {
	c := &subsetChecks{pass: pass, parents: make(map[ast.Node]ast.Node)}
	for _, file := range pass.Files {
		var stack []ast.Node
		ast.Inspect(file, func(n ast.Node) bool {
			if n == nil {
				stack = stack[:len(stack)-1]
				return false
			}
			if len(stack) > 0 {
				c.parents[n] = stack[len(stack)-1]
			}
			stack = append(stack, n)
			return true
		})
	}
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool { c.checkOwnedNode(n); return true })
	}
}

// checkOwnedNode verifies slot placement and value uses.
func (c *subsetChecks) checkOwnedNode(node ast.Node) {
	switch n := node.(type) {
	case *ast.FuncLit:
		c.checkSlotCapture(n)
	case *ast.FuncType:
		if c.hasSlotValue(c.pass.TypesInfo.TypeOf(n)) {
			c.pass.Reportf(n.Pos(), "function types cannot pass or return Owned slots or holders by value")
		}
	case *ast.FuncDecl:
		c.checkSlotFunction(n)
	case *ast.TypeSpec:
		if !n.Assign.IsValid() && checker.IsLibraryType(c.pass.TypesInfo.TypeOf(n.Type), "Owned") {
			c.pass.Reportf(n.Pos(), "defined copies of Owned slots are unsupported")
		}
	case *ast.ArrayType, *ast.MapType, *ast.ChanType:
		if expression, ok := node.(ast.Expr); ok && c.hasSlotValue(c.pass.TypesInfo.TypeOf(expression)) {
			c.pass.Reportf(node.Pos(), "containers of Owned slots or holders are unsupported")
		}
	case *ast.StarExpr:
		if c.pass.TypesInfo.Types[n].IsType() && checker.IsLibraryType(c.pass.TypesInfo.TypeOf(n.X), "Owned") {
			c.pass.Reportf(n.Pos(), "pointers to Owned slots are unsupported")
		}
	case *ast.ValueSpec:
		c.checkSlotDeclaration(n)
	case *ast.StructType:
		c.checkSlotFields(n)
	case *ast.CallExpr:
		if c.ownedSlotCall(n) {
			c.checkOwnedCall(n)
		}
	}
	if expression, ok := node.(ast.Expr); ok {
		c.checkSlotExpression(expression)
	}
}

// checkSlotFunction rejects holder receivers and signature copies.
func (c *subsetChecks) checkSlotFunction(n *ast.FuncDecl) {
	sig, _ := c.pass.TypesInfo.TypeOf(n.Name).(*types.Signature)
	if sig == nil {
		return
	}
	if (sig.Recv() != nil && c.hasSlotValue(sig.Recv().Type())) ||
		c.slotTuple(sig.Params()) || c.slotTuple(sig.Results()) {
		c.pass.Reportf(n.Pos(), "Owned slots and their holders cannot be passed or returned by value")
	}
}

// checkSlotDeclaration permits standalone slots only at package scope.
func (c *subsetChecks) checkSlotDeclaration(n *ast.ValueSpec) {
	for _, name := range n.Names {
		variable, ok := c.pass.TypesInfo.Defs[name].(*types.Var)
		if ok && checker.IsLibraryType(variable.Type(), "Owned") && variable.Parent() != c.pass.Pkg.Scope() {
			c.pass.Reportf(name.Pos(), "Owned slots require a field or package variable")
		}
	}
}

// checkSlotFields requires slots to be direct named fields.
func (c *subsetChecks) checkSlotFields(n *ast.StructType) {
	structure, _ := c.pass.TypesInfo.TypeOf(n).Underlying().(*types.Struct)
	if structure == nil {
		return
	}
	for field := range structure.Fields() {
		if c.hasSlotValue(field.Type()) && (!checker.IsLibraryType(field.Type(), "Owned") || field.Embedded()) {
			c.pass.Reportf(n.Pos(), "Owned slots require direct named fields; nested slot storage is unsupported")
		}
	}
}

// checkSlotExpression rejects copies and unproved pointer transport.
func (c *subsetChecks) checkSlotExpression(expression ast.Expr) {
	if c.pass.TypesInfo.Types[expression].IsType() {
		return
	}
	if id, ok := expression.(*ast.Ident); ok {
		if selector, ok := c.parents[id].(*ast.SelectorExpr); ok && selector.Sel == id {
			return
		}
		if c.pass.TypesInfo.Defs[id] != nil {
			return
		}
		switch object := c.pass.TypesInfo.ObjectOf(id).(type) {
		case *types.TypeName, *types.Func:
			return
		case *types.Var:
			if object.IsField() {
				return
			}
		}
	}
	typ := c.pass.TypesInfo.TypeOf(expression)
	if c.holderPointer(typ) && !c.allowedHolderPointer(expression) {
		c.pass.Reportf(expression.Pos(), "Owned holder pointers require local bindings or direct typed calls")
		return
	}
	if c.hasSlotValue(typ) && !c.allowedSlotValue(expression) {
		c.pass.Reportf(expression.Pos(), "Owned slots and their holders cannot be copied or transported by value")
	}
}

// checkSlotCapture rejects slot and holder references from an outer scope.
func (c *subsetChecks) checkSlotCapture(literal *ast.FuncLit) {
	functionScope := c.pass.TypesInfo.Scopes[literal.Type]
	ast.Inspect(literal.Body, func(node ast.Node) bool {
		name, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		variable, ok := c.pass.TypesInfo.Uses[name].(*types.Var)
		if !ok || variable.IsField() || !c.slotRelevant(variable.Type()) {
			return true
		}
		for scope := variable.Parent(); scope != nil; scope = scope.Parent() {
			if scope == functionScope {
				return true
			}
		}
		c.pass.Reportf(name.Pos(), "closures cannot capture Owned slots or holders")
		return true
	})
}

// slotTuple rejects value parameters and results containing a slot.
func (c *subsetChecks) slotTuple(tuple *types.Tuple) bool {
	for i := 0; i < tuple.Len(); i++ {
		if c.hasSlotValue(tuple.At(i).Type()) {
			return true
		}
	}
	return false
}

// holderPointer identifies a pointer to a slot-bearing struct.
func (c *subsetChecks) holderPointer(t types.Type) bool {
	if t == nil {
		return false
	}
	pointer, ok := t.Underlying().(*types.Pointer)
	if !ok || checker.IsLibraryType(pointer.Elem(), "Owned") {
		return false
	}
	_, structure := pointer.Elem().Underlying().(*types.Struct)
	return structure && c.hasSlotValue(pointer.Elem())
}

// allowedHolderPointer permits local pointers, field selection, and checked direct calls.
func (c *subsetChecks) allowedHolderPointer(expression ast.Expr) bool {
	parent := c.parents[expression]
	if paren, ok := parent.(*ast.ParenExpr); ok {
		return c.allowedHolderPointer(paren)
	}
	switch node := parent.(type) {
	case *ast.SelectorExpr:
		return node.X == expression && checker.IsLibraryType(c.pass.TypesInfo.TypeOf(node), "Owned")
	case *ast.StarExpr:
		selector, ok := c.parents[node].(*ast.SelectorExpr)
		return ok && selector.X == node && checker.IsLibraryType(c.pass.TypesInfo.TypeOf(selector), "Owned")
	case *ast.AssignStmt:
		if len(node.Lhs) != 1 || len(node.Rhs) != 1 || node.Rhs[0] != expression {
			return false
		}
		name, ok := node.Lhs[0].(*ast.Ident)
		return ok && c.localHolderBinding(name, expression)
	case *ast.ValueSpec:
		if len(node.Names) != 1 || len(node.Values) != 1 || node.Values[0] != expression {
			return false
		}
		return c.localHolderBinding(node.Names[0], expression)
	case *ast.CallExpr:
		return c.directHolderArgument(node, expression)
	}
	return false
}

// localHolderBinding checks the destination of a holder pointer copy.
func (c *subsetChecks) localHolderBinding(name *ast.Ident, expression ast.Expr) bool {
	variable, ok := c.pass.TypesInfo.ObjectOf(name).(*types.Var)
	return ok && variable.Parent() != c.pass.Pkg.Scope() &&
		types.Identical(variable.Type(), c.pass.TypesInfo.TypeOf(expression))
}

// directHolderArgument requires a non-generic function with an exact pointer parameter.
func (c *subsetChecks) directHolderArgument(call *ast.CallExpr, expression ast.Expr) bool {
	function, ok := c.calledObject(call.Fun).(*types.Func)
	if !ok || genericSignature(function.Origin().Type().(*types.Signature)) {
		return false
	}
	signature := function.Type().(*types.Signature)
	if signature.Recv() != nil {
		return false
	}
	for index, argument := range call.Args {
		if argument == expression {
			parameter := callParameter(signature, index, call.Ellipsis.IsValid())
			return types.Identical(c.pass.TypesInfo.TypeOf(expression), parameter)
		}
	}
	return false
}

// allowedSlotValue admits only a slot receiver or a fresh holder address.
func (c *subsetChecks) allowedSlotValue(expression ast.Expr) bool {
	parent := c.parents[expression]
	if paren, ok := parent.(*ast.ParenExpr); ok {
		return c.allowedSlotValue(paren)
	}
	if checker.IsLibraryType(c.pass.TypesInfo.TypeOf(expression), "Owned") {
		selector, ok := parent.(*ast.SelectorExpr)
		return ok && selector.X == expression && c.ownedSlotSelector(selector) && c.directCall(selector) != nil
	}
	if selector, ok := parent.(*ast.SelectorExpr); ok && selector.X == expression &&
		checker.IsLibraryType(c.pass.TypesInfo.TypeOf(selector), "Owned") {
		return true
	}
	if _, ok := ast.Unparen(expression).(*ast.CompositeLit); ok {
		unary, ok := parent.(*ast.UnaryExpr)
		return ok && unary.Op == token.AND
	}
	return false
}

// checkOwnedCall proves the complete Store and Load statement forms.
func (c *subsetChecks) checkOwnedCall(call *ast.CallExpr) {
	selector := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !c.validSlotReceiver(selector.X) {
		c.pass.Reportf(call.Pos(), "Owned calls require a direct field or package slot")
		return
	}
	switch selector.Sel.Name {
	case "Store":
		if len(call.Args) != 1 || !c.freshSlotArgument(call.Args[0]) {
			c.pass.Reportf(call.Pos(), "Owned.Store requires a fresh LocalRef allocation")
			return
		}
		if _, ok := c.unparenthesizedParent(call).(*ast.ExprStmt); !ok {
			c.pass.Reportf(call.Pos(), "Owned.Store requires a direct synchronous statement")
		}
	case "Load":
		if !c.localLoadBinding(call) {
			c.pass.Reportf(call.Pos(), "Owned.Load requires a single local pointer binding")
		}
	}
}

// localLoadBinding requires one local destination for an atomic load.
func (c *subsetChecks) localLoadBinding(call *ast.CallExpr) bool {
	var name *ast.Ident
	switch statement := c.unparenthesizedParent(call).(type) {
	case *ast.AssignStmt:
		if len(statement.Lhs) != 1 || len(statement.Rhs) != 1 {
			return false
		}
		name, _ = statement.Lhs[0].(*ast.Ident)
	case *ast.ValueSpec:
		if len(statement.Names) != 1 || len(statement.Values) != 1 {
			return false
		}
		name = statement.Names[0]
	}
	if name == nil || name.Name == "_" {
		return false
	}
	variable, ok := c.pass.TypesInfo.ObjectOf(name).(*types.Var)
	return ok && variable.Parent() != c.pass.Pkg.Scope()
}

// validSlotReceiver excludes slot pointers and temporary slots.
func (c *subsetChecks) validSlotReceiver(receiver ast.Expr) bool {
	receiver = ast.Unparen(receiver)
	if !checker.IsLibraryType(c.pass.TypesInfo.TypeOf(receiver), "Owned") {
		return false
	}
	switch value := receiver.(type) {
	case *ast.Ident:
		variable, ok := c.pass.TypesInfo.ObjectOf(value).(*types.Var)
		return ok && variable.Parent() == c.pass.Pkg.Scope()
	case *ast.SelectorExpr:
		selection := c.pass.TypesInfo.Selections[value]
		if selection == nil {
			variable, ok := c.pass.TypesInfo.ObjectOf(value.Sel).(*types.Var)
			return ok && variable.Pkg() != nil && variable.Parent() == variable.Pkg().Scope()
		}
		return selection != nil && selection.Kind() == types.FieldVal && len(selection.Index()) == 1 &&
			c.pass.TypesInfo.Types[value].Addressable()
	}
	return false
}

// freshSlotArgument permits only a composite allocation or new(T).
func (c *subsetChecks) freshSlotArgument(expression ast.Expr) bool {
	switch value := ast.Unparen(expression).(type) {
	case *ast.UnaryExpr:
		_, literal := ast.Unparen(value.X).(*ast.CompositeLit)
		return value.Op == token.AND && literal
	case *ast.CallExpr:
		builtin, ok := c.calledObject(value.Fun).(*types.Builtin)
		return ok && builtin.Name() == "new" && len(value.Args) == 1 && c.pass.TypesInfo.Types[value.Args[0]].IsType()
	}
	return false
}
