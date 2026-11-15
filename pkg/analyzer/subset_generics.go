package analyzer

import (
	"go/ast"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// genericInstance checks one source instance outside unchecked dependency type arguments.
func (c *subsetChecks) genericInstance(name *ast.Ident) {
	instance, found := c.pass.TypesInfo.Instances[name]
	if !found {
		return
	}
	object := c.pass.TypesInfo.Uses[name]
	if libraryObject(object) || object != nil && object.Pkg() != nil && object.Pkg().Path() == checker.RuntimePath {
		return
	}
	for i := 0; i < instance.TypeArgs.Len(); i++ {
		arg := instance.TypeArgs.At(i)
		if c.queries.Transport(arg) || c.slotRelevant(arg) || c.queries.Callable(arg) {
			c.report(name, "user generic instantiations with immutable types are unsupported")
		}
	}
}

// genericStorage checks complete declared types and constraints; bodies use normal source checks.
func (c *subsetChecks) genericStorage(name *ast.Ident) {
	typ := c.pass.TypesInfo.Defs[name].Type()
	declared := []types.Type{typ}
	var parameters *types.TypeParamList
	switch definition := typ.(type) {
	case *types.Signature:
		parameters = definition.TypeParams()
		if definition.Recv() != nil {
			declared = append(declared, definition.Recv().Type())
		}
		for parameter := range definition.RecvTypeParams().Len() {
			declared = append(declared, definition.RecvTypeParams().At(parameter))
		}
	case *types.Named:
		if checker.Marked(definition) || checker.FrozenInterface(definition) {
			c.report(name, "user generic declarations with immutable storage are unsupported")
			return
		}
		parameters = definition.TypeParams()
	case *types.Alias:
		parameters = definition.TypeParams()
	}
	for parameter := range parameters.Len() {
		declared = append(declared, parameters.At(parameter))
	}
	for _, typ := range declared {
		if c.queries.Transport(typ) || c.slotRelevant(typ) || c.queries.Callable(typ) {
			c.report(name, "user generic declarations with immutable storage are unsupported")
			return
		}
	}
}
