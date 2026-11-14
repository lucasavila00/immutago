package analyzer

import (
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
