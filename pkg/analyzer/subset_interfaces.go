package analyzer

import (
	"go/ast"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// matchFrozenInterface proves every value placed in a sealed interface.
// Exact method receivers exclude promoted calls and implicit dereferences.
func (c *subsetChecks) matchFrozenInterface(source, destination types.Type) checker.Failure {
	if reason := c.proofs.FrozenFailure(destination); reason.Reason != "" {
		return reason
	}
	if source == types.Typ[types.UntypedNil] {
		return checker.Failure{}
	}
	if reason := c.proofs.FrozenFailure(source); reason.Reason != "" {
		reason.Reason = "Frozen interface implementation: " + reason.Reason
		return reason
	}
	if checker.FrozenInterface(source) {
		return checker.Failure{}
	}
	target := types.Unalias(source)
	if pointer, ok := target.(*types.Pointer); ok {
		target = types.Unalias(pointer.Elem())
	}
	if _, named := target.(*types.Named); !named || !checker.Frozen(target) {
		return shapeFailure("Frozen interfaces require concrete Frozen record implementations")
	}
	methods := types.NewMethodSet(source)
	for method := range destination.Underlying().(*types.Interface).Methods() {
		if checker.FrozenInterfaceSeal(method) {
			continue
		}
		selection := methods.Lookup(method.Pkg(), method.Name())
		if selection == nil || len(selection.Index()) != 1 {
			return shapeFailure("Frozen interface implementations cannot use promoted methods")
		}
		signature := selection.Obj().Type().(*types.Signature)
		if !types.Identical(source, signature.Recv().Type()) {
			return shapeFailure("Frozen interface method receivers must match the converted type exactly")
		}
	}
	return checker.Failure{}
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
func (c *subsetChecks) matchFrozenInterfaceMethod(selector *ast.SelectorExpr) checker.Failure {
	typ := c.pass.TypesInfo.TypeOf(selector.X)
	if reason := c.proofs.FrozenFailure(typ); reason.Reason != "" {
		return reason
	}
	selection := c.pass.TypesInfo.Selections[selector]
	if len(selection.Index()) != 1 || !types.Identical(typ, selection.Recv()) {
		return shapeFailure("Frozen interface methods require a direct interface receiver")
	}
	return checker.Failure{}
}
