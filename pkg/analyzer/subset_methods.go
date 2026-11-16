package analyzer

import (
	"go/ast"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// provedMethodReceiver checks the shared type rules without restricting method type parameters.
func (c *subsetChecks) provedMethodReceiver(signature *types.Signature) bool {
	if signature.Recv() == nil {
		return false
	}
	receiver := types.Unalias(signature.Recv().Type())
	if pointer, ok := receiver.(*types.Pointer); ok {
		receiver = types.Unalias(pointer.Elem())
	}
	named, ok := receiver.(*types.Named)
	return ok && (named.TypeParams().Len() == 0 || checker.IsHandle(named)) &&
		(c.proofs.CopiedReason(named) == "" || checker.Marked(named) && c.proofs.Reason(named) == "")
}

// selectedMethodReceiver follows the compiler's embedded fields to the receiver storage.
func selectedMethodReceiver(selection *types.Selection) types.Type {
	receiver := selection.Recv()
	for _, index := range selection.Index()[:len(selection.Index())-1] {
		if pointer, ok := receiver.Underlying().(*types.Pointer); ok {
			receiver = pointer.Elem()
		}
		receiver = receiver.Underlying().(*types.Struct).Field(index).Type()
	}
	return receiver
}

// matchMethod checks receiver shapes and restricted calls.
// Persistent receivers also permit method values and expressions with the same shape.
func (c *subsetChecks) matchMethod(selector *ast.SelectorExpr) checker.Failure {
	selection := c.pass.TypesInfo.Selections[selector]
	restricted := c.queries.Transport(selection.Recv()) || c.queries.Callable(c.pass.TypesInfo.TypeOf(selector))
	if selection.Kind() != types.MethodVal && restricted {
		return shapeFailure("method expressions require ordinary or persistent Frozen receivers and contracts")
	}
	if c.directCall(selector) == nil && restricted {
		return shapeFailure("method values require ordinary or persistent Frozen receivers and contracts")
	}
	if _, parameter := types.Unalias(selection.Recv()).(*types.TypeParam); parameter && !restricted {
		return checker.Failure{}
	}
	if _, dynamic := selection.Recv().Underlying().(*types.Interface); dynamic {
		if checker.FrozenInterface(selection.Recv()) {
			return c.matchFrozenInterfaceMethod(selector)
		}
		if !restricted {
			return checker.Failure{}
		}
		return shapeFailure("dynamic calls require ordinary or persistent Frozen parameters and results")
	}
	signature := selection.Obj().(*types.Func).Origin().Type().(*types.Signature)
	// Promoted methods must not take the address of an embedded Frozen value.
	if checker.FrozenPointer(signature.Recv().Type()) && !checker.FrozenPointer(selectedMethodReceiver(selection)) {
		return shapeFailure("immutable method receivers must match their declared type exactly")
	}
	if !c.protectedStorage(selector.X) && !c.queries.Owns(signature.Recv().Type()) {
		return checker.Failure{}
	}
	if len(selection.Index()) != 1 {
		return shapeFailure("promoted immutable methods are unsupported")
	}
	if !c.provedMethodReceiver(signature) {
		return shapeFailure("immutable methods require a proved marked or Copied receiver")
	}
	signature = selection.Obj().Type().(*types.Signature)
	if !types.Identical(c.pass.TypesInfo.TypeOf(selector.X), signature.Recv().Type()) {
		return shapeFailure("immutable method receivers must match their declared type exactly")
	}
	return checker.Failure{}
}
