package analyzer

import (
	"go/ast"
	"go/types"
)

// slotPointerMethod permits a holder method selection that saves only its address.
func (c *subsetChecks) slotPointerMethod(selector *ast.SelectorExpr) bool {
	selection := c.pass.TypesInfo.Selections[selector]
	if selection == nil || selection.Kind() != types.MethodVal {
		return false
	}
	method := selection.Obj().(*types.Func).Origin().Type().(*types.Signature)
	_, pointer := method.Recv().Type().Underlying().(*types.Pointer)
	return pointer
}

// checkSlotSelection rejects the implicit copy made by a holder value receiver.
func (c *subsetChecks) checkSlotSelection(selector *ast.SelectorExpr) {
	selection := c.pass.TypesInfo.Selections[selector]
	if selection == nil || selection.Kind() == types.FieldVal {
		return
	}
	method := selection.Obj().(*types.Func).Origin().Type().(*types.Signature)
	if c.hasSlotValue(method.Recv().Type()) {
		c.pass.Reportf(selector.Pos(), "Cell slots and their holders cannot be copied by a value receiver")
	}
}
