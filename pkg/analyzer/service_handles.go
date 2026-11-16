package analyzer

import (
	"go/ast"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// checkHandles proves Ref targets and checks direct Borrow calls.
func (c *subsetChecks) checkHandles() {
	for _, file := range c.pass.Files {
		ast.Inspect(file, c.visitHandle)
	}
}

// visitHandle checks reference types and direct access to the runtime accessor.
func (c *subsetChecks) visitHandle(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.Ident:
		if instance, ok := c.pass.TypesInfo.Instances[node]; ok {
			c.checkReference(node, instance.Type)
		}
	case *ast.SelectorExpr:
		if selection := c.pass.TypesInfo.Selections[node]; selection != nil && refAccessor(selection.Obj()) {
			if rule := c.matchBorrow(node); rule.Reason != "" {
				c.reportFailure(node, rule)
			}
		}
	}
	if expression, ok := node.(ast.Expr); ok {
		if value := c.pass.TypesInfo.Types[expression]; value.IsValue() {
			c.checkReference(expression, value.Type)
		}
	}
	return true
}

// checkReference proves the concrete service target without restricting value transport.
func (c *subsetChecks) checkReference(node ast.Node, typ types.Type) {
	if checker.IsRuntimeType(typ, "Ref") {
		if reason := c.proofs.Failure(typ); reason.Reason != "" {
			c.reportFailure(node, reason)
		}
	}
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
func (c *subsetChecks) matchBorrow(selector *ast.SelectorExpr) checker.Failure {
	const rejected = "use a service reference only as ref.Borrow().Method(args); " +
		"do not extract the accessor or service pointer"
	selection := c.pass.TypesInfo.Selections[selector]
	if selection == nil || selection.Kind() != types.MethodVal || len(selection.Index()) != 1 {
		return shapeFailure(rejected)
	}
	current := c.directCall(selector)
	if current == nil {
		return shapeFailure(rejected)
	}
	message, ok := c.unparenthesizedParent(current).(*ast.SelectorExpr)
	if !ok || ast.Unparen(message.X) != current {
		return shapeFailure(rejected)
	}
	method := c.pass.TypesInfo.Selections[message]
	if method == nil || method.Kind() != types.MethodVal || c.directCall(message) == nil ||
		!method.Obj().Exported() || runtimeObject(method.Obj()) ||
		method.Obj().Name() == "Init" || method.Obj().Name() == "Run" {
		return shapeFailure(rejected)
	}
	if !checker.Service(method.Recv()) {
		return shapeFailure("Ref requires a service target that embeds supervise.Service")
	}
	return checker.Failure{}
}
