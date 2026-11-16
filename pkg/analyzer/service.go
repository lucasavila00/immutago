package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// checkServiceEmbedding checks the selected runtime method path before startup.
func (c *checks) checkServiceEmbedding(pos token.Pos, t types.Type) {
	if checker.Service(t) {
		if reason := checker.ServiceEmbeddingReason(t); reason != "" {
			c.report(pos, reason)
		}
	}
}

// checkServiceResults checks all user-defined exported methods, including promoted methods.
func (c *checks) checkServiceResults(pos token.Pos, t types.Type) {
	if !checker.Service(t) {
		return
	}
	methods := types.NewMethodSet(types.NewPointer(t))
	for i := 0; i < methods.Len(); i++ {
		method := methods.At(i).Obj()
		if !serviceMethod(method) {
			continue
		}
		at := pos
		if method.Pkg() == c.pass.Pkg && method.Pos().IsValid() {
			at = method.Pos()
		}
		signature := methods.At(i).Type().(*types.Signature)
		for result := range signature.Results().Variables() {
			if types.Identical(result.Type(), types.Universe.Lookup("error").Type()) {
				continue
			}
			if reason := c.proofs.ReadResultReason(result.Type()); reason != "" {
				c.report(at, "service method "+method.Name()+" must return a read result: "+reason+
					"\n  result type: "+diagnosticType(result.Type())+"\n  fix: "+structuralFix(result.Type(), reason))
			}
		}
	}
}

// serviceMethod identifies exported application methods by their compiler-resolved object.
func serviceMethod(method types.Object) bool {
	return method.Exported() && method.Name() != "Init" && method.Name() != "Run" && !runtimeObject(method)
}

// runtimeObject identifies the trusted runtime API that can transport immutable values.
func runtimeObject(object types.Object) bool {
	return object != nil && object.Pkg() != nil && object.Pkg().Path() == checker.RuntimePath
}

// checkServiceStorage proves protected arguments before the runtime saves them.
func (c *checks) checkServiceStorage(name *ast.Ident, argument types.Type) {
	if !c.queries.Owns(argument) && !c.queries.Callable(argument) {
		return
	}
	if reason := c.proofs.FrozenReason(argument); reason != "" {
		c.report(name.Pos(), "stored service argument: "+reason+
			"\n  fix: pass Frozen data or a service Ref; Local pointers cannot be stored")
	}
}
