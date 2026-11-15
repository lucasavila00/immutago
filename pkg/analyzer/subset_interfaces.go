package analyzer

import (
	"go/ast"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// matchFrozenInterface proves every value placed in a sealed interface.
// Exact method receivers exclude promoted calls and implicit dereferences.
func (c *subsetChecks) matchFrozenInterface(source, destination types.Type) string {
	if reason := c.proofs.FrozenReason(destination); reason != "" {
		return reason
	}
	if source == types.Typ[types.UntypedNil] {
		return ""
	}
	if reason := c.proofs.FrozenReason(source); reason != "" {
		return "Frozen interface implementation: " + reason
	}
	if checker.FrozenInterface(source) {
		return ""
	}
	target := types.Unalias(source)
	if pointer, ok := target.(*types.Pointer); ok {
		target = types.Unalias(pointer.Elem())
	}
	if _, named := target.(*types.Named); !named || !checker.Frozen(target) {
		return "Frozen interfaces require concrete Frozen record implementations"
	}
	methods := types.NewMethodSet(source)
	for method := range destination.Underlying().(*types.Interface).Methods() {
		if checker.FrozenInterfaceSeal(method) {
			continue
		}
		selection := methods.Lookup(method.Pkg(), method.Name())
		if selection == nil || len(selection.Index()) != 1 {
			return "Frozen interface implementations cannot use promoted methods"
		}
		signature := selection.Obj().Type().(*types.Signature)
		if !types.Identical(source, signature.Recv().Type()) {
			return "Frozen interface method receivers must match the converted type exactly"
		}
	}
	return ""
}

// frozenInterfaceMethodType identifies declared signatures inside a sealed contract.
func (c *subsetChecks) frozenInterfaceMethodType(expression ast.Expr) bool {
	field, ok := c.parents[expression].(*ast.Field)
	if !ok || field.Type != expression || len(field.Names) != 1 {
		return false
	}
	list, ok := c.parents[field].(*ast.FieldList)
	if !ok {
		return false
	}
	contract, ok := c.parents[list].(*ast.InterfaceType)
	return ok && checker.FrozenInterface(c.pass.TypesInfo.TypeOf(contract))
}

// matchFrozenInterfaceMethod requires a direct call through a proved named contract.
func (c *subsetChecks) matchFrozenInterfaceMethod(selector *ast.SelectorExpr) string {
	typ := c.pass.TypesInfo.TypeOf(selector.X)
	if reason := c.proofs.FrozenReason(typ); reason != "" {
		return reason
	}
	selection := c.pass.TypesInfo.Selections[selector]
	if len(selection.Index()) != 1 || !types.Identical(typ, selection.Recv()) {
		return "Frozen interface methods require a direct interface receiver"
	}
	return ""
}
