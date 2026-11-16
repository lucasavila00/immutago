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
	errorType := types.Universe.Lookup("error").Type()
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
		for result := range signature.Results().Variables() {
			if types.Identical(result.Type(), errorType) {
				continue
			}
			reason := c.proofs.ReadResultReason(result.Type())
			if reason == "" {
				continue
			}
			fix := structuralFix(result.Type(), reason)
			if contract, ok := result.Type().Underlying().(*types.Interface); ok &&
				types.Implements(contract, errorType.Underlying().(*types.Interface)) {
				fix = "declare the result as error or a type alias such as type Failure = error"
			}
			c.report(at, "service method "+method.Name()+" must return a read result or error: "+reason+
				"\n  result type: "+diagnosticType(result.Type())+"\n  fix: "+fix)
		}
	}
}

// runtimeObject identifies the trusted runtime API that can transport immutable values.
func runtimeObject(object types.Object) bool {
	return object != nil && object.Pkg() != nil && object.Pkg().Path() == checker.RuntimePath
}

// visitService checks service boundaries in packages that do not use im data.
func (c *checks) visitService(node ast.Node) bool {
	switch declaration := node.(type) {
	case *ast.TypeSpec:
		if object := c.pass.TypesInfo.Defs[declaration.Name]; object != nil {
			c.checkServiceResults(declaration.Name.Pos(), object.Type())
		}
	case *ast.FuncDecl:
		c.checkFunctionBody(declaration)
	case *ast.StructType:
		c.checkServiceEmbedding(declaration.Pos(), c.pass.TypesInfo.TypeOf(declaration))
		c.checkServiceResults(declaration.Pos(), c.pass.TypesInfo.TypeOf(declaration))
	}
	return true
}

// checkServiceStorage applies im storage rules when the runtime saves protected arguments.
func (c *checks) checkServiceStorage(name *ast.Ident, argument types.Type) {
	if !c.queries.Owns(argument) && !c.queries.Callable(argument) {
		return
	}
	if reason := c.proofs.FrozenReason(argument); reason != "" {
		c.report(name.Pos(), "stored im argument: "+reason+
			"\n  fix: pass an immutable value or a Frozen pointer; Local pointers cannot be stored")
	}
}
