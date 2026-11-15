package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
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
	return checker.IsLibraryType(receiver, "Cell")
}

// ownedSlotCall recognizes a direct call on the sealed slot type.
func (c *subsetChecks) ownedSlotCall(call *ast.CallExpr) bool {
	selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	return ok && c.ownedSlotSelector(selector)
}

// containsSlotValue finds slots in value storage, without following pointers.
func (c *subsetChecks) containsSlotValue(t types.Type, seen map[types.Type]bool) bool {
	if t == nil || seen[t] {
		return false
	}
	if result, ok := c.slots[t]; ok {
		return result
	}
	seen[t] = true
	if checker.IsLibraryType(t, "Cell") {
		return true
	}
	switch typ := t.Underlying().(type) {
	case *types.Array:
		return c.containsSlotValue(typ.Elem(), seen)
	case *types.Slice:
		return c.containsSlotValue(typ.Elem(), seen)
	case *types.Map:
		return c.containsSlotValue(typ.Key(), seen) || c.containsSlotValue(typ.Elem(), seen)
	case *types.Chan:
		return c.containsSlotValue(typ.Elem(), seen)
	case *types.Signature:
		return c.containsSlotValue(typ.Params(), seen) || c.containsSlotValue(typ.Results(), seen)
	case *types.Tuple:
		return c.tupleHasSlotValue(typ, seen)
	case *types.Struct:
		return c.structHasSlotValue(typ, seen)
	case *types.Interface:
		return c.interfaceHasSlotValue(typ, seen)
	case *types.Union:
		for index := 0; index < typ.Len(); index++ {
			if c.containsSlotValue(typ.Term(index).Type(), seen) {
				return true
			}
		}
	}
	return false
}

// interfaceHasSlotValue checks embedded constraints and method contracts.
func (c *subsetChecks) interfaceHasSlotValue(contract *types.Interface, seen map[types.Type]bool) bool {
	for index := 0; index < contract.NumEmbeddeds(); index++ {
		if c.containsSlotValue(contract.EmbeddedType(index), seen) {
			return true
		}
	}
	for index := 0; index < contract.NumMethods(); index++ {
		if c.containsSlotValue(contract.Method(index).Type(), seen) {
			return true
		}
	}
	return false
}

// tupleHasSlotValue checks each parameter or result type.
func (c *subsetChecks) tupleHasSlotValue(tuple *types.Tuple, seen map[types.Type]bool) bool {
	for i := 0; i < tuple.Len(); i++ {
		if c.containsSlotValue(tuple.At(i).Type(), seen) {
			return true
		}
	}
	return false
}

// structHasSlotValue checks each field stored by value.
func (c *subsetChecks) structHasSlotValue(structure *types.Struct, seen map[types.Type]bool) bool {
	for field := range structure.Fields() {
		if c.containsSlotValue(field.Type(), seen) {
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
	seen := make(map[types.Type]bool)
	result := c.containsSlotValue(t, seen)
	if !result {
		// Cache negatives only after the complete query, not at a cycle's back edge.
		for visited := range seen {
			c.slots[visited] = false
		}
	}
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

// checkCellNode verifies slot placement and value uses.
func (c *subsetChecks) checkCellNode(node ast.Node) {
	c.checkCellTypeNode(node)
	switch n := node.(type) {
	case *ast.FuncDecl:
		c.checkSlotFunction(n)
	case *ast.ValueSpec:
		c.checkSlotDeclaration(n)
	case *ast.StructType:
		c.checkSlotFields(n)
	case *ast.CallExpr:
		if c.ownedSlotCall(n) {
			c.checkCellCall(n)
		} else {
			c.checkHolderMethodCall(n)
		}
	}
	if expression, ok := node.(ast.Expr); ok {
		c.checkSlotExpression(expression)
	}
}

// checkCellTypeNode rejects unsupported type-level holder transport.
func (c *subsetChecks) checkCellTypeNode(node ast.Node) {
	switch n := node.(type) {
	case *ast.InterfaceType:
		if c.hasHolderReference(c.pass.TypesInfo.TypeOf(n)) {
			c.pass.Reportf(n.Pos(), "interfaces cannot contain Cell holder pointers")
		}
	case *ast.FuncType:
		c.checkCellSignatureType(n)
	case *ast.TypeSpec:
		c.checkCellTypeSpec(n)
	case *ast.ArrayType, *ast.MapType, *ast.ChanType:
		expression, _ := node.(ast.Expr)
		typ := c.pass.TypesInfo.TypeOf(expression)
		if c.hasSlotValue(typ) || c.hasHolderReference(typ) {
			c.pass.Reportf(node.Pos(), "containers of Cell slots or holders are unsupported")
		}
	case *ast.StarExpr:
		c.checkCellPointerType(n)
	}
}

// checkCellSignatureType rejects callback contracts carrying holder pointers.
func (c *subsetChecks) checkCellSignatureType(n *ast.FuncType) {
	if _, direct := c.parents[n].(*ast.FuncDecl); !direct {
		if signature, ok := c.pass.TypesInfo.TypeOf(n).(*types.Signature); ok {
			c.checkHolderSignature(signature, false, n)
		}
	}
	if c.hasSlotValue(c.pass.TypesInfo.TypeOf(n)) {
		c.pass.Reportf(n.Pos(), "function types cannot pass or return Cell slots or holders by value")
	}
}

// checkCellTypeSpec rejects new defined slot and holder pointer types.
func (c *subsetChecks) checkCellTypeSpec(n *ast.TypeSpec) {
	if n.Assign.IsValid() {
		return
	}
	typ := c.pass.TypesInfo.TypeOf(n.Type)
	if c.holderPointer(typ) {
		c.pass.Reportf(n.Pos(), "defined holder pointer types are unsupported")
	}
	if checker.IsLibraryType(typ, "Cell") {
		c.pass.Reportf(n.Pos(), "defined copies of Cell slots are unsupported")
	}
}

// checkCellPointerType rejects slot addresses and nested holder pointers.
func (c *subsetChecks) checkCellPointerType(n *ast.StarExpr) {
	if !c.pass.TypesInfo.Types[n].IsType() {
		return
	}
	element := c.pass.TypesInfo.TypeOf(n.X)
	if checker.IsLibraryType(element, "Cell") || c.hasHolderReference(element) {
		c.pass.Reportf(n.Pos(), "pointers to Cell slots or holder references are unsupported")
	}
}

// checkSlotFunction rejects holder receivers and signature copies.
func (c *subsetChecks) checkSlotFunction(n *ast.FuncDecl) {
	sig, _ := c.pass.TypesInfo.TypeOf(n.Name).(*types.Signature)
	if sig == nil {
		return
	}
	c.checkHolderSignature(sig, true, n)
	if (sig.Recv() != nil && c.hasSlotValue(sig.Recv().Type())) ||
		c.slotTuple(sig.Params()) || c.slotTuple(sig.Results()) {
		c.pass.Reportf(n.Pos(), "Cell slots and their holders cannot be passed or returned by value")
	}
}

// checkSlotDeclaration permits standalone slots only at package scope.
func (c *subsetChecks) checkSlotDeclaration(n *ast.ValueSpec) {
	for _, name := range n.Names {
		variable, ok := c.pass.TypesInfo.Defs[name].(*types.Var)
		if !ok {
			continue
		}
		if checker.IsLibraryType(variable.Type(), "Cell") && variable.Parent() != c.pass.Pkg.Scope() {
			c.pass.Reportf(name.Pos(), "Cell slots require a field or package variable")
		}
		if variable.Parent() == c.pass.Pkg.Scope() && c.hasHolderReference(variable.Type()) {
			c.pass.Reportf(name.Pos(), "Cell holder pointers cannot be stored in package variables")
		}
	}
}

// checkSlotFields permits only direct named slots and nested holder values.
func (c *subsetChecks) checkSlotFields(n *ast.StructType) {
	structure, _ := c.pass.TypesInfo.TypeOf(n).Underlying().(*types.Struct)
	if structure == nil {
		return
	}
	for field := range structure.Fields() {
		if c.hasHolderReference(field.Type()) {
			c.pass.Reportf(n.Pos(), "Cell holder pointers cannot be stored in fields")
		}
		if c.hasSlotValue(field.Type()) {
			_, record := field.Type().Underlying().(*types.Struct)
			if field.Embedded() || !record {
				c.pass.Reportf(n.Pos(), "Cell slots and holders require direct named fields")
			}
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
		case *types.TypeName, *types.Builtin:
			return
		case *types.Var:
			if object.IsField() {
				return
			}
		}
	}
	typ := c.pass.TypesInfo.TypeOf(expression)
	if typ == nil {
		return
	}
	if _, callable := typ.Underlying().(*types.Signature); callable &&
		c.hasHolderReference(typ) && c.directCall(expression) == nil {
		c.pass.Reportf(expression.Pos(), "function values cannot carry Cell holder pointers")
		return
	}
	if c.holderPointer(typ) && (!c.validHolderSource(expression) || !c.allowedHolderPointer(expression)) {
		c.pass.Reportf(expression.Pos(), "Cell holder pointers require local bindings or direct typed calls")
		return
	}
	if c.hasSlotValue(typ) && !c.allowedSlotValue(expression) {
		c.pass.Reportf(expression.Pos(), "Cell slots and their holders cannot be copied or transported by value")
	}
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
	if !ok || checker.IsLibraryType(pointer.Elem(), "Cell") || checker.Service(pointer.Elem()) {
		return false
	}
	_, structure := pointer.Elem().Underlying().(*types.Struct)
	return structure && c.hasSlotValue(pointer.Elem())
}

// validHolderSource permits local bindings and fresh holder allocation.
func (c *subsetChecks) validHolderSource(expression ast.Expr) bool {
	switch source := expression.(type) {
	case *ast.ParenExpr:
		return true
	case *ast.Ident:
		variable, ok := c.pass.TypesInfo.ObjectOf(source).(*types.Var)
		return ok && variable.Parent() != variable.Pkg().Scope()
	case *ast.UnaryExpr:
		_, literal := ast.Unparen(source.X).(*ast.CompositeLit)
		return source.Op == token.AND && literal
	case *ast.CallExpr:
		builtin, ok := c.calledObject(source.Fun).(*types.Builtin)
		return ok && builtin.Name() == "new" && len(source.Args) == 1 &&
			c.pass.TypesInfo.Types[source.Args[0]].IsType()
	}
	return false
}

// allowedHolderPointer permits local pointers, field selection, and checked direct calls.
func (c *subsetChecks) allowedHolderPointer(expression ast.Expr) bool {
	parent := c.parents[expression]
	if paren, ok := parent.(*ast.ParenExpr); ok {
		return c.allowedHolderPointer(paren)
	}
	switch node := parent.(type) {
	case *ast.SelectorExpr:
		return node.X == expression && (c.hasSlotValue(c.pass.TypesInfo.TypeOf(node)) || c.matchHolderMethod(node) == "")
	case *ast.StarExpr:
		selector, ok := c.unparenthesizedParent(node).(*ast.SelectorExpr)
		return ok && ast.Unparen(selector.X) == node && c.hasSlotValue(c.pass.TypesInfo.TypeOf(selector))
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
		if _, deferred := c.parents[node].(*ast.DeferStmt); deferred {
			return false
		}
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
			if signature.Variadic() && index >= signature.Params().Len()-1 {
				return false
			}
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
	if checker.IsLibraryType(c.pass.TypesInfo.TypeOf(expression), "Cell") {
		selector, ok := parent.(*ast.SelectorExpr)
		return ok && selector.X == expression && c.ownedSlotSelector(selector) && c.directCall(selector) != nil
	}
	if selector, ok := parent.(*ast.SelectorExpr); ok && selector.X == expression {
		return c.hasSlotValue(c.pass.TypesInfo.TypeOf(selector)) || c.matchHolderMethod(selector) == ""
	}
	if _, ok := ast.Unparen(expression).(*ast.CompositeLit); ok {
		unary, ok := parent.(*ast.UnaryExpr)
		return ok && unary.Op == token.AND
	}
	return false
}

// checkCellCall checks the slot capability; payloads use the general Frozen proof.
func (c *subsetChecks) checkCellCall(call *ast.CallExpr) {
	selector := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if selector.Sel.Name == "LoadSharedPointer" && !checker.FrozenPointer(c.pass.TypesInfo.TypeOf(call)) {
		c.pass.Reportf(call.Pos(), "Cell.LoadSharedPointer requires a concrete Frozen record")
		return
	}
	if !c.validSlotReceiver(selector.X) {
		c.pass.Reportf(call.Pos(), "Cell calls require a direct field or package slot")
		return
	}
	if selector.Sel.Name == "Store" {
		if _, ok := c.unparenthesizedParent(call).(*ast.ExprStmt); !ok {
			c.pass.Reportf(call.Pos(), "Cell.Store requires a direct synchronous statement")
			return
		}
	}
	if rule := c.matchStatement(call); rule != "" {
		c.report(c.statementOperation(call), rule)
	}
}

// validSlotReceiver excludes slot pointers and temporary slots.
func (c *subsetChecks) validSlotReceiver(receiver ast.Expr) bool {
	receiver = ast.Unparen(receiver)
	if !checker.IsLibraryType(c.pass.TypesInfo.TypeOf(receiver), "Cell") {
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
