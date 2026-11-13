package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
	"golang.org/x/tools/go/analysis"
)

// subsetChecks rejects transports outside the local, synchronous borrow contract.
type subsetChecks struct {
	pass     *analysis.Pass
	parents  map[ast.Node]ast.Node
	exempt   map[ast.Node]bool
	reported map[token.Pos]bool
	owners   map[*types.Var]bool
}

// checkSubset checks the supported borrow boundary with compiler-resolved types.
func checkSubset(pass *analysis.Pass, owners map[*types.Var]bool) {
	c := &subsetChecks{pass: pass, parents: make(map[ast.Node]ast.Node),
		owners: owners, exempt: make(map[ast.Node]bool), reported: make(map[token.Pos]bool)}
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
	c.ownedInputs()
	for _, file := range pass.Files {
		ast.Inspect(file, c.visit)
	}
	for id, instance := range pass.TypesInfo.Instances {
		obj := pass.TypesInfo.Uses[id]
		if _, ok := obj.(*types.Func); !ok || libraryObject(obj) {
			continue
		}
		for i := 0; i < instance.TypeArgs.Len(); i++ {
			if checker.Contains(instance.TypeArgs.At(i)) {
				c.report(id, "generic calls with immutable types are unsupported")
			}
		}
	}
}

// report emits one boundary failure per source position.
func (c *subsetChecks) report(n ast.Node, message string) {
	if !c.reported[n.Pos()] {
		c.reported[n.Pos()] = true
		c.pass.Reportf(n.Pos(), "unsupported immutable borrow: %s%s\n  fix: %s",
			message, sourceContext(c.pass, n), subsetFix(message))
	}
}

// visit checks source operations and their inferred types.
func (c *subsetChecks) visit(n ast.Node) bool {
	c.statementBoundary(n)
	if expression, ok := n.(ast.Expr); ok {
		c.expressionType(expression)
		c.freezeFunctionUse(expression)
		c.synchronousStorage(expression)
		c.statementShape(expression)
	}

	switch x := n.(type) {
	case *ast.TypeAssertExpr:
		if checker.Contains(c.pass.TypesInfo.TypeOf(x)) {
			c.report(x, "immutable interface assertions are unsupported")
		}
	case *ast.UnaryExpr:
		if x.Op == token.AND {
			c.address(x)
		}
	case *ast.SliceExpr:
		if !stringType(c.pass.TypesInfo.TypeOf(x.X)) && c.protectedRoot(x.X) {
			c.report(x, "slice views of immutable storage are unsupported")
		}
	case *ast.CallExpr:
		c.call(x)
	case *ast.FuncLit:
		c.closure(x)
	case *ast.SelectorExpr:
		c.methodValue(x)
	case *ast.AssignStmt:
		c.assignment(x)
	case *ast.ValueSpec:
		c.declaration(x)
	case *ast.ReturnStmt:
		c.returnedValues(x)
	case *ast.SendStmt:
		if channel, ok := c.pass.TypesInfo.TypeOf(x.Chan).Underlying().(*types.Chan); ok {
			c.transport(x.Value, channel.Elem())
		}
	}
	if expression, ok := n.(ast.Expr); ok {
		c.referenceUse(expression)
	}
	return true
}

// subsetType rejects hidden references and native containers, including inferred result types.
func subsetType(t types.Type, seen map[types.Type]bool) string {
	if t == nil || seen[t] || checker.IsCollection(t) {
		return ""
	}
	seen[t] = true
	switch u := t.Underlying().(type) {
	case *types.Pointer:
		if _, ok := u.Elem().Underlying().(*types.Pointer); ok && checker.Contains(t) {
			return "pointer indirection is unsupported"
		}
		return subsetType(u.Elem(), seen)
	case *types.Array:
		if protectedReference(u.Elem()) {
			return "arrays of shared references are unsupported"
		}
		return subsetType(u.Elem(), seen)
	case *types.Signature:
		return subsetResults(u.Results())
	case *types.Tuple:
		return subsetTuple(u, seen)
	default:
		return subsetContainer(u)
	}
}

// protectedReference identifies a pointer whose target contains immutable storage.
func protectedReference(t types.Type) bool {
	if t == nil {
		return false
	}
	_, ok := t.Underlying().(*types.Pointer)
	return ok && checker.Contains(t)
}

// protectedRoot follows projections to detect scalar fields of immutable owners.
func (c *subsetChecks) protectedRoot(e ast.Expr) bool {
	if checker.Contains(c.pass.TypesInfo.TypeOf(e)) {
		return true
	}
	switch x := e.(type) {
	case *ast.SelectorExpr:
		return c.protectedRoot(x.X)
	case *ast.IndexExpr:
		return c.protectedRoot(x.X)
	case *ast.ParenExpr:
		return c.protectedRoot(x.X)
	case *ast.StarExpr:
		return c.protectedRoot(x.X)
	}
	return false
}

// address permits only a whole local owner as the source of a shared borrow.
func (c *subsetChecks) address(e *ast.UnaryExpr) {
	if !c.protectedRoot(e.X) {
		return
	}
	target := ast.Unparen(e.X)
	if !checker.Contains(c.pass.TypesInfo.TypeOf(target)) {
		c.report(e, "scalar interior addresses are unsupported")
		return
	}
	for {
		selection, ok := target.(*ast.SelectorExpr)
		if !ok {
			break
		}
		target = ast.Unparen(selection.X)
	}
	id, ok := target.(*ast.Ident)
	if !ok {
		c.report(e, "only whole local values and marked fields can be borrowed")
		return
	}

	if protectedReference(c.pass.TypesInfo.TypeOf(id)) {
		c.report(e, "borrowing through a shared reference is unsupported")
		return
	}
	obj := c.pass.TypesInfo.ObjectOf(id)
	if obj == nil || obj.Parent() == c.pass.Pkg.Scope() {
		c.report(e, "only whole local values can be borrowed")
	}
}

// libraryObject identifies the fixed library by package identity.
func libraryObject(obj types.Object) bool {
	return obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == checker.LibraryPath
}

// stringType permits immutable string slices even when the string belongs to a marked owner.
func stringType(t types.Type) bool {
	basic, ok := t.Underlying().(*types.Basic)
	return ok && basic.Info()&types.IsString != 0
}

// assignment checks immutable transport in AssignStmt.
func (c *subsetChecks) assignment(x *ast.AssignStmt) {
	if len(x.Rhs) == 1 {
		var destinations []types.Type
		for _, lhs := range x.Lhs {
			destinations = append(destinations, c.pass.TypesInfo.TypeOf(lhs))
		}
		c.tupleTransport(x.Rhs[0], destinations)
	}
	for i, rhs := range x.Rhs {
		if i < len(x.Lhs) {
			c.transport(rhs, c.pass.TypesInfo.TypeOf(x.Lhs[i]))
			c.globalStore(x.Lhs[i], rhs)
		}
	}
}

// declaration checks immutable transport in ValueSpec.
func (c *subsetChecks) declaration(x *ast.ValueSpec) {
	if len(x.Values) == 1 {
		var destinations []types.Type
		for _, id := range x.Names {
			if obj := c.pass.TypesInfo.Defs[id]; obj != nil {
				destinations = append(destinations, obj.Type())
			}
		}
		c.tupleTransport(x.Values[0], destinations)
	}
	for i, id := range x.Names {
		obj := c.pass.TypesInfo.Defs[id]
		if obj == nil {
			continue
		}
		if protectedReference(obj.Type()) && obj.Parent() == c.pass.Pkg.Scope() {
			c.report(id, "global shared references are unsupported")
		}
		if i < len(x.Values) {
			c.transport(x.Values[i], obj.Type())
		}
	}
}

// returnedValues checks immutable transport in ReturnStmt.
func (c *subsetChecks) returnedValues(x *ast.ReturnStmt) {
	if len(x.Results) == 1 {
		if tuple, ok := c.pass.TypesInfo.TypeOf(x.Results[0]).(*types.Tuple); ok {
			var destinations []types.Type
			for i := 0; i < tuple.Len(); i++ {
				destinations = append(destinations, c.returnType(x, i))
			}
			c.tupleTransport(x.Results[0], destinations)
		}
	}
	for i, e := range x.Results {
		if destination := c.returnType(x, i); destination != nil {
			c.transport(e, destination)
		}
		if protectedReference(c.pass.TypesInfo.TypeOf(e)) {
			c.report(e, "returning shared references is unsupported")
		}
	}
}

// subsetContainer checks the three native container forms excluded from immutable storage.
func subsetContainer(t types.Type) string {
	switch u := t.(type) {
	case *types.Slice:
		if checker.Contains(u.Elem()) {
			return "native slices of immutable storage are unsupported"
		}
	case *types.Map:
		if checker.Contains(u.Key()) || checker.Contains(u.Elem()) {
			return "native maps of immutable storage are unsupported"
		}
	case *types.Chan:
		if checker.Contains(u.Elem()) {
			return "channels of immutable storage are unsupported"
		}
	}
	return ""
}

// subsetResults rejects escaping shared references in function results.
func subsetResults(results *types.Tuple) string {
	for i := 0; i < results.Len(); i++ {
		if protectedReference(results.At(i).Type()) {
			return "returning shared references is unsupported"
		}
	}
	return ""
}

// subsetTuple checks each inferred result type without losing tuple context.
func subsetTuple(tuple *types.Tuple, seen map[types.Type]bool) string {
	for i := 0; i < tuple.Len(); i++ {
		if reason := subsetType(tuple.At(i).Type(), seen); reason != "" {
			return reason
		}
	}
	return ""
}

// expressionType checks resolved types while admitting native builders checked by ownership analysis.
func (c *subsetChecks) expressionType(expression ast.Expr) {
	t := c.pass.TypesInfo.TypeOf(expression)
	if t == nil || c.exempt[expression] || c.ownedExpression(expression) {
		return
	}
	if reason := subsetType(t, make(map[types.Type]bool)); reason != "" {
		c.report(expression, reason)
	}
}
