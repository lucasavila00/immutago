package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// checkOpaqueValue permits the Value method of im.Opaque only in the package that declares its type argument.
func (c *checks) checkOpaqueValue(selector *ast.SelectorExpr) {
	selection := c.pass.TypesInfo.Selections[selector]
	if selection == nil || selection.Kind() == types.FieldVal || !libraryObject(selection.Obj()) ||
		selection.Obj().Name() != "Value" {
		return
	}
	receiver := selection.Recv()
	if pointer, ok := receiver.Underlying().(*types.Pointer); ok {
		receiver = pointer.Elem()
	}
	if !checker.IsOpaque(receiver) {
		return
	}
	c.checkTrustedPackage(selector.Sel.Pos(), types.Unalias(receiver).(*types.Named).TypeArgs().At(0), "Opaque.Value")
}

// checkTrustedPackage requires a named type argument that the current package declares.
func (c *checks) checkTrustedPackage(pos token.Pos, argument types.Type, use string) {
	named, ok := types.Unalias(argument).(*types.Named)
	if ok && named.Obj().Pkg() == c.pass.Pkg {
		return
	}
	c.report(pos, use+" requires a type that this package declares\n  type: "+diagnosticType(argument)+
		"\n  rule: only the package that declares T can wrap or read an im.Opaque[T]"+
		"\n  fix: call a function of the package that declares the type")
}
