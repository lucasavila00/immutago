package analyzer

import (
	"go/ast"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// markedMethodReceiver proves the complete concrete receiver type of a user method.
func (c *subsetChecks) markedMethodReceiver(signature *types.Signature) bool {
	if signature.Recv() == nil || genericSignature(signature) {
		return false
	}
	receiver := types.Unalias(signature.Recv().Type())
	if pointer, ok := receiver.(*types.Pointer); ok {
		receiver = types.Unalias(pointer.Elem())
	}
	named, ok := receiver.(*types.Named)
	return ok && named.TypeParams().Len() == 0 && (checker.Marked(named) || checker.Copied(named)) &&
		c.proofs.Reason(named) == ""
}

// matchMethod requires direct concrete calls for protected arguments and results.
// Protected receivers also require an exact, non-promoted immutable receiver.
func (c *subsetChecks) matchMethod(selector *ast.SelectorExpr) string {
	selection := c.pass.TypesInfo.Selections[selector]
	if selection.Kind() != types.MethodVal {
		return "immutable method expressions are unsupported"
	}
	if c.directCall(selector) == nil {
		return "bound immutable methods are unsupported"
	}
	if _, dynamic := selection.Recv().Underlying().(*types.Interface); dynamic {
		if checker.FrozenInterface(selection.Recv()) {
			return c.matchFrozenInterfaceMethod(selector)
		}
		return "dynamic calls with immutable storage are unsupported"
	}
	signature := selection.Obj().(*types.Func).Origin().Type().(*types.Signature)
	if !c.protectedStorage(selector.X) && !c.immutableObject(signature.Recv().Type()) {
		if genericSignature(signature) && !runtimeObject(selection.Obj()) {
			return "user generic instantiations with immutable types are unsupported"
		}
		return ""
	}
	if len(selection.Index()) != 1 {
		return "promoted immutable methods are unsupported"
	}
	if !c.markedMethodReceiver(signature) {
		return "immutable methods require a concrete non-generic marked receiver"
	}
	if !types.Identical(c.pass.TypesInfo.TypeOf(selector.X), signature.Recv().Type()) {
		return "immutable method receivers must match their declared type exactly"
	}
	return ""
}
