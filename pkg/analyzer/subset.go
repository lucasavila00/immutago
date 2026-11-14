package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
	"golang.org/x/tools/go/analysis"
)

// subsetChecks matches immutable operations and rejects unsupported transport.
type subsetChecks struct {
	slots      map[types.Type]bool
	holderRefs map[types.Type]bool
	queries    checker.Classifier
	proofs     *checker.Prover
	pass       *analysis.Pass
	parents    map[ast.Node]ast.Node
	exempt     map[ast.Node]bool
	reported   map[token.Pos]bool
	owners     map[*types.Var]bool
	literals   map[*ast.CompositeLit]string
	processes  map[types.Type]bool
}

// checkSubset checks the supported immutable forms with compiler-resolved types.
func checkSubset(pass *analysis.Pass, owners map[*types.Var]bool, proofs *checker.Prover) {
	c := &subsetChecks{pass: pass, proofs: proofs, parents: make(map[ast.Node]ast.Node),
		owners: owners, exempt: make(map[ast.Node]bool), reported: make(map[token.Pos]bool),
		literals: make(map[*ast.CompositeLit]string)}
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
}

// report emits one boundary failure per source position.
func (c *subsetChecks) report(n ast.Node, message string) {
	if !c.reported[n.Pos()] {
		c.reported[n.Pos()] = true
		c.pass.Reportf(n.Pos(), "unsupported immutable operation: %s%s\n  fix: %s",
			message, sourceContext(c.pass, n), subsetFix(message))
	}
}

// visit sends each protected operation through the complete source-form matcher.
func (c *subsetChecks) visit(node ast.Node) bool {
	if expression, ok := node.(ast.Expr); ok && c.pass.TypesInfo.Types[expression].IsType() &&
		checker.IsRuntimeType(c.pass.TypesInfo.TypeOf(expression), checker.UncheckedDependencyType) {
		return false
	}
	c.checkOwnerNode(node)
	c.checkProcessNode(node)
	c.declarationNode(node)
	if name, ok := node.(*ast.Ident); ok {
		c.genericInstance(name)
	}
	expression, ok := node.(ast.Expr)
	if !ok {
		return true
	}
	c.expressionType(expression)
	if c.operationExpression(expression) {
		if operation, rule := c.matchOperation(expression); rule != "" {
			c.report(operation, rule)
		}
	}
	return !uncheckedTypeArguments(c.pass.TypesInfo, expression)
}

// subsetType rejects hidden references and native containers, including inferred result types.
func (c *subsetChecks) subsetType(t types.Type, seen map[types.Type]bool) string {
	if t == nil || seen[t] || checker.IsCollection(t) {
		return ""
	}
	seen[t] = true
	switch u := t.Underlying().(type) {
	case *types.Pointer:
		if _, ok := u.Elem().Underlying().(*types.Pointer); ok && c.queries.Contains(t) {
			return "pointer indirection is unsupported"
		}
		if c.queries.Contains(t) && !checker.Marked(u.Elem()) {
			return "immutable references require a concrete marked struct target"
		}
		if checker.Marked(u.Elem()) {
			if reason := c.proofs.Reason(u.Elem()); reason != "" {
				return reason
			}
		}
		return c.subsetType(u.Elem(), seen)
	case *types.Array:
		if c.queries.Contains(t) {
			if reason := c.proofs.Reason(t); reason != "" {
				return reason
			}
		}
		return c.subsetType(u.Elem(), seen)
	case *types.Signature:
		return subsetResults(u.Results())
	case *types.Tuple:
		return c.subsetTuple(u, seen)
	default:
		return c.subsetContainer(u)
	}
}

// protectedReference identifies a pointer whose target contains immutable storage.
func (c *subsetChecks) protectedReference(t types.Type) bool {
	if t == nil {
		return false
	}
	_, ok := t.Underlying().(*types.Pointer)
	return ok && c.queries.Contains(t)
}

// protectedRoot follows projections to detect scalar fields of immutable owners.
func (c *subsetChecks) protectedRoot(e ast.Expr) bool {
	if c.queries.Contains(c.pass.TypesInfo.TypeOf(e)) {
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

// libraryObject identifies the fixed library by package identity.
func libraryObject(obj types.Object) bool {
	return obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == checker.LibraryPath
}

// stringType permits immutable string slices even when the string belongs to a marked owner.
func stringType(t types.Type) bool {
	basic, ok := t.Underlying().(*types.Basic)
	return ok && basic.Info()&types.IsString != 0
}

// subsetContainer checks the three native container forms excluded from immutable storage.
func (c *subsetChecks) subsetContainer(t types.Type) string {
	switch u := t.(type) {
	case *types.Slice:
		if c.queries.Contains(u.Elem()) {
			return "native slices of immutable storage are unsupported"
		}
	case *types.Map:
		if c.queries.Contains(u.Key()) || c.queries.Contains(u.Elem()) {
			return "native maps of immutable storage are unsupported"
		}
	case *types.Chan:
		if c.queries.Contains(u.Elem()) {
			return "channels of immutable storage are unsupported"
		}
	}
	return ""
}

// subsetResults rejects local-only references in function results.
func subsetResults(results *types.Tuple) string {
	for i := 0; i < results.Len(); i++ {
		if localReference(results.At(i).Type()) {
			return "returning Scoped pointers is unsupported"
		}
	}
	return ""
}

// subsetTuple checks each inferred result type without losing tuple context.
func (c *subsetChecks) subsetTuple(tuple *types.Tuple, seen map[types.Type]bool) string {
	for i := 0; i < tuple.Len(); i++ {
		if reason := c.subsetType(tuple.At(i).Type(), seen); reason != "" {
			return reason
		}
	}
	return ""
}

// expressionType checks resolved types while admitting native builders checked by ownership analysis.
func (c *subsetChecks) expressionType(expression ast.Expr) {
	// A builtin signature describes this call, not a user function's return boundary.
	if name, ok := expression.(*ast.Ident); ok {
		if c.pass.TypesInfo.Defs[name] != nil {
			return
		}
		if _, builtin := c.pass.TypesInfo.ObjectOf(name).(*types.Builtin); builtin {
			return
		}
	}
	if c.directSlotMethodExpression(expression) {
		return
	}
	t := c.pass.TypesInfo.TypeOf(expression)
	if c.pass.TypesInfo.Types[expression].IsType() && c.queries.Callable(t) && !c.trustedFunctionType(expression) {
		if _, declaration := c.parents[expression].(*ast.FuncDecl); !declaration {
			c.report(expression, "immutable callbacks are unsupported")
		}
	}
	if storedScopedReference(t, make(map[types.Type]bool)) {
		c.report(expression, "containers of Scoped pointers are unsupported")
	}
	if t == nil || c.exempt[expression] || c.ownedExpression(expression) {
		return
	}
	if reason := c.subsetType(t, make(map[types.Type]bool)); reason != "" {
		c.report(expression, reason)
	}
}

// directSlotMethodExpression identifies method syntax in a checked slot call.
func (c *subsetChecks) directSlotMethodExpression(expression ast.Expr) bool {
	selector, ok := expression.(*ast.SelectorExpr)
	if !ok {
		id, isName := expression.(*ast.Ident)
		if !isName {
			return false
		}
		selector, ok = c.parents[id].(*ast.SelectorExpr)
		if !ok || selector.Sel != id {
			return false
		}
	}
	return c.ownedSlotSelector(selector) && c.directCall(selector) != nil
}

// storedScopedReference checks native container elements even when the handle has an ownership proof.
func storedScopedReference(t types.Type, seen map[types.Type]bool) bool {
	if t == nil || seen[t] {
		return false
	}
	seen[t] = true
	var elements []types.Type
	switch storage := t.Underlying().(type) {
	case *types.Array:
		elements = []types.Type{storage.Elem()}
	case *types.Slice:
		elements = []types.Type{storage.Elem()}
	case *types.Map:
		elements = []types.Type{storage.Key(), storage.Elem()}
	case *types.Chan:
		elements = []types.Type{storage.Elem()}
	}
	for _, element := range elements {
		if localReference(element) || storedScopedReference(element, seen) {
			return true
		}
	}
	return false
}
