package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// checkServiceResults checks all user-defined exported methods, including promoted methods.
func (c *checks) checkServiceResults(pos token.Pos, t types.Type) {
	if !checker.Service(t) {
		return
	}
	methods := types.NewMethodSet(types.NewPointer(t))
	for i := 0; i < methods.Len(); i++ {
		method := methods.At(i).Obj()
		if !method.Exported() || method.Name() == "Init" || method.Name() == "Run" || runtimeObject(method) {
			continue
		}
		at := pos
		if method.Pkg() == c.pass.Pkg && method.Pos().IsValid() {
			at = method.Pos()
		}
		signature := methods.At(i).Type().(*types.Signature)
		if signature.Results().Len() != 1 {
			c.report(at, "service method "+method.Name()+" must return exactly one Scoped result")
			continue
		}
		result := signature.Results().At(0).Type()
		if reason := c.proofs.ScopedResultReason(result); reason != "" {
			c.report(at, "service method "+method.Name()+" must return a Scoped result: "+reason+
				"\n  result type: "+diagnosticType(result))
		}
	}
}

// runtimeObject identifies the trusted runtime API that can transport immutable values.
func runtimeObject(object types.Object) bool {
	return object != nil && object.Pkg() != nil && object.Pkg().Path() == checker.RuntimePath
}

// visitService checks method results in packages that do not use im data.
func (c *checks) visitService(node ast.Node) bool {
	if declaration, ok := node.(*ast.TypeSpec); ok {
		if object := c.pass.TypesInfo.Defs[declaration.Name]; object != nil {
			c.checkServiceResults(declaration.Name.Pos(), object.Type())
		}
	}
	return true
}

// checkServiceStorage applies im storage rules when the runtime saves protected arguments.
func (c *checks) checkServiceStorage(name *ast.Ident, argument types.Type) {
	if !c.queries.Contains(argument) && !c.queries.Callable(argument) {
		return
	}
	if reason := c.proofs.Reason(argument); reason != "" {
		c.report(name.Pos(), "stored im argument: "+reason+
			"\n  fix: pass an immutable value or a Shared pointer; Scoped pointers cannot be stored")
	}
}
