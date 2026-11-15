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

// matchMethod checks receiver shapes and restricted calls.
// Persistent receivers also permit method values and expressions with the same shape.
func (c *subsetChecks) matchMethod(selector *ast.SelectorExpr) string {
	selection := c.pass.TypesInfo.Selections[selector]
	restricted := c.queries.Transport(selection.Recv()) || c.queries.Callable(c.pass.TypesInfo.TypeOf(selector))
	if selection.Kind() != types.MethodVal && restricted {
		return "immutable method expressions are unsupported"
	}
	if c.directCall(selector) == nil && restricted {
		return "bound immutable methods are unsupported"
	}
	if _, parameter := types.Unalias(selection.Recv()).(*types.TypeParam); parameter && !restricted {
		return ""
	}
	if _, dynamic := selection.Recv().Underlying().(*types.Interface); dynamic {
		if checker.FrozenInterface(selection.Recv()) {
			return c.matchFrozenInterfaceMethod(selector)
		}
		return "dynamic calls with immutable storage are unsupported"
	}
	signature := selection.Obj().(*types.Func).Origin().Type().(*types.Signature)
	if !c.protectedStorage(selector.X) && !c.queries.Contains(signature.Recv().Type()) {
		if genericSignature(signature) && !runtimeObject(selection.Obj()) {
			return "user generic instantiations with immutable types are unsupported"
		}
		return ""
	}
	if len(selection.Index()) != 1 {
		return "promoted immutable methods are unsupported"
	}
	if runtimeObject(selection.Obj()) {
		signature = selection.Obj().Type().(*types.Signature)
	} else if !c.markedMethodReceiver(signature) {
		return "immutable methods require a concrete non-generic marked receiver"
	}
	if !types.Identical(c.pass.TypesInfo.TypeOf(selector.X), signature.Recv().Type()) {
		return "immutable method receivers must match their declared type exactly"
	}
	return ""
}
