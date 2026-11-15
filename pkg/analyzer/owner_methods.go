package analyzer

import (
	"go/ast"
	"go/types"
)

// slotMethodSelection permits methods that do not copy a slot-bearing receiver.
func (c *subsetChecks) slotMethodSelection(selector *ast.SelectorExpr) bool {
	selection := c.pass.TypesInfo.Selections[selector]
	if selection == nil || selection.Kind() != types.MethodVal {
		return false
	}
	method := selection.Obj().(*types.Func).Origin().Type().(*types.Signature)
	return !c.hasSlotValue(method.Recv().Type())
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
