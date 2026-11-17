package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// checkServiceLifecycle checks declared receivers on named and anonymous service types.
func (c *checks) checkServiceLifecycle(node ast.Node, stack []ast.Node) {
	var t types.Type
	switch declaration := node.(type) {
	case *ast.TypeSpec:
		t = c.pass.TypesInfo.Defs[declaration.Name].Type()
	case *ast.StructType:
		parent := stack[len(stack)-1]
		for i := len(stack) - 2; i >= 0; i-- {
			if _, ok := parent.(*ast.ParenExpr); !ok {
				break
			}
			parent = stack[i]
		}
		if _, ok := parent.(*ast.TypeSpec); ok {
			return // The named type's method set includes its declared overrides.
		}
		t = c.pass.TypesInfo.TypeOf(declaration)
	default:
		return
	}
	if !checker.Service(t) {
		return
	}
	methods := types.NewMethodSet(types.NewPointer(t))
	for _, name := range []string{"Init", "Run"} {
		selection := methods.Lookup(nil, name)
		if selection == nil {
			continue // Start's Go constraint requires both method signatures.
		}
		method := selection.Obj()
		receiver := method.Type().(*types.Signature).Recv().Type()
		if _, ok := types.Unalias(receiver).(*types.Pointer); ok {
			continue
		}
		c.report(node.Pos(), "service lifecycle method "+name+" must have a declared pointer receiver"+
			"\n  service type: "+diagnosticType(t)+"; declared receiver: "+diagnosticType(receiver)+
			"\n  fix: declare a pointer receiver method on the service; forward the call if needed")
	}
}

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
			failure := c.proofs.ReadResultFailure(result.Type())
			if failure.Reason == "" {
				continue
			}
			fix := failure.Repair
			if contract, ok := result.Type().Underlying().(*types.Interface); ok &&
				types.Implements(contract, errorType.Underlying().(*types.Interface)) {
				fix = "declare the result as Go's predeclared error or an alias such as type Failure = error; " +
					"rename any declaration that shadows error"
			}
			c.report(at, "service method "+method.Name()+" must return a read result or error: "+failure.Reason+
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
	if failure := c.proofs.FrozenFailure(argument); failure.Reason != "" {
		c.report(name.Pos(), "start argument: "+failure.Reason+"\n  fix: "+failure.Repair)
	}
}
