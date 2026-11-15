package analyzer

import (
	"go/ast"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// checkHandles checks Borrow and each use of stored service references in selected packages.
func (c *subsetChecks) checkHandles() {
	for _, file := range c.pass.Files {
		ast.Inspect(file, c.visitHandle)
	}
}

// visitHandle checks type instances, Borrow calls, and consumers of service references.
func (c *subsetChecks) visitHandle(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.Ident:
		c.handleInstance(node)
	case *ast.SelectorExpr:
		if selection := c.pass.TypesInfo.Selections[node]; selection != nil && refAccessor(selection.Obj()) {
			if rule := c.matchBorrow(node); rule != "" {
				c.report(node, rule)
			}
		}
	}
	expression, ok := node.(ast.Expr)
	if !ok || !c.handleOperand(expression) {
		return true
	}
	if typ := c.pass.TypesInfo.TypeOf(expression); checker.IsRuntimeType(typ, "Ref") {
		if reason := c.proofs.Reason(typ); reason != "" {
			c.report(expression, reason)
		}
	}
	if rule := c.matchHandleUse(expression); rule != "" {
		c.report(expression, rule)
	}
	return true
}

// refAccessor identifies the runtime accessor even through aliases and promoted methods.
func refAccessor(object types.Object) bool {
	if !runtimeObject(object) || object.Name() != "Borrow" {
		return false
	}
	signature, ok := object.Type().(*types.Signature)
	return ok && signature.Recv() != nil && checker.IsRuntimeType(signature.Recv().Type(), "Ref")
}

// matchBorrow accepts only a direct exported method call on the selected service.
func (c *subsetChecks) matchBorrow(selector *ast.SelectorExpr) string {
	const rejected = "use a service reference only as ref.Borrow().Method(args); " +
		"do not extract the accessor or service pointer"
	selection := c.pass.TypesInfo.Selections[selector]
	if selection == nil || selection.Kind() != types.MethodVal || len(selection.Index()) != 1 {
		return rejected
	}
	current := c.directCall(selector)
	if current == nil {
		return rejected
	}
	message, ok := c.unparenthesizedParent(current).(*ast.SelectorExpr)
	if !ok || ast.Unparen(message.X) != current {
		return rejected
	}
	method := c.pass.TypesInfo.Selections[message]
	if method == nil || method.Kind() != types.MethodVal || c.directCall(message) == nil ||
		!method.Obj().Exported() || runtimeObject(method.Obj()) ||
		method.Obj().Name() == "Init" || method.Obj().Name() == "Run" {
		return rejected
	}
	if !checker.Service(method.Recv()) {
		return "Ref requires a service target that embeds supervise.Service"
	}
	return ""
}

// handleOperand separates actual stored values from type syntax, names, and selector metadata.
func (c *subsetChecks) handleOperand(expression ast.Expr) bool {
	if c.pass.TypesInfo.Types[expression].IsType() {
		return false
	}
	if selector, ok := c.parents[expression].(*ast.SelectorExpr); ok && selector.Sel == expression {
		return false
	}
	if name, ok := expression.(*ast.Ident); ok && c.pass.TypesInfo.Defs[name] != nil {
		return false
	}
	t := c.pass.TypesInfo.TypeOf(expression)
	if tuple, ok := t.(*types.Tuple); ok {
		for variable := range tuple.Variables() {
			if c.queries.Handles(variable.Type()) {
				return true
			}
		}
	}
	return c.queries.Handles(t)
}

// handleInstance keeps service references out of unchecked user generic bodies and constraints.
func (c *subsetChecks) handleInstance(name *ast.Ident) {
	instance, ok := c.pass.TypesInfo.Instances[name]
	if !ok {
		return
	}
	if checker.IsRuntimeType(instance.Type, "Ref") {
		if reason := c.proofs.Reason(instance.Type); reason != "" {
			c.report(name, reason)
		}
	}
	object := c.pass.TypesInfo.Uses[name]
	if runtimeObject(object) || libraryObject(object) {
		return
	}
	for i := 0; i < instance.TypeArgs.Len(); i++ {
		if c.queries.Handles(instance.TypeArgs.At(i)) {
			c.report(name, "user generic type arguments cannot contain service references; "+
				"use concrete Ref parameters and storage")
			return
		}
	}
}
