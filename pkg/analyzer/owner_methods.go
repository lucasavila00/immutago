package analyzer

import (
	"go/ast"
	"go/types"
)

// holderMethod identifies a declared method on a concrete Cell holder.
func (c *subsetChecks) holderMethod(selector *ast.SelectorExpr) *types.Signature {
	selection := c.pass.TypesInfo.Selections[selector]
	if selection == nil || selection.Kind() == types.FieldVal {
		return nil
	}
	method, ok := selection.Obj().(*types.Func)
	if !ok {
		return nil
	}
	signature := method.Origin().Type().(*types.Signature)
	if signature.Recv() == nil || !c.holderPointer(signature.Recv().Type()) {
		return nil
	}
	return signature
}

// matchHolderMethod permits direct pointer methods without copying the holder.
func (c *subsetChecks) matchHolderMethod(selector *ast.SelectorExpr) string {
	const rejected = "Cell holder methods require a direct concrete pointer receiver"
	signature := c.holderMethod(selector)
	selection := c.pass.TypesInfo.Selections[selector]
	if signature == nil || selection.Kind() != types.MethodVal || len(selection.Index()) != 1 ||
		genericSignature(signature) || c.directCall(selector) == nil {
		return rejected
	}
	receiver := c.pass.TypesInfo.TypeOf(selector.X)
	if types.Identical(receiver, signature.Recv().Type()) && c.validHolderSource(selector.X) {
		return ""
	}
	pointer := signature.Recv().Type().Underlying().(*types.Pointer)
	if !types.Identical(receiver, pointer.Elem()) || !c.pass.TypesInfo.Types[selector.X].Addressable() {
		return rejected
	}
	switch source := ast.Unparen(selector.X).(type) {
	case *ast.Ident:
		if _, variable := c.pass.TypesInfo.ObjectOf(source).(*types.Var); variable {
			return ""
		}
	case *ast.SelectorExpr:
		field := c.pass.TypesInfo.Selections[source]
		if field == nil || field.Kind() == types.FieldVal && len(field.Index()) == 1 {
			return ""
		}
	}
	return rejected
}

// checkHolderMethodCall checks all holder calls, including those with scalar results.
func (c *subsetChecks) checkHolderMethodCall(call *ast.CallExpr) {
	selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok || c.holderMethod(selector) == nil {
		return
	}
	if rule := c.matchHolderMethod(selector); rule != "" {
		c.report(call, rule)
		return
	}
	if rule := c.matchStatement(call); rule != "" {
		c.report(c.statementOperation(call), rule)
	}
}
