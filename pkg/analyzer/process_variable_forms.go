package analyzer

import (
	"go/ast"
	"go/types"
)

var scalarAtomics = map[string]bool{"Bool": true, "Int32": true, "Int64": true, "Uint32": true, "Uint64": true}

// atomicType identifies a named type of package sync/atomic.
func atomicType(t types.Type) bool {
	named, ok := types.Unalias(t).(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == "sync/atomic"
}

// atomicReason accepts an unexported scalar atomic of this package that the package uses only as the receiver
// of direct method calls.
func (c *checks) atomicReason(variable *types.Var) string {
	name := types.Unalias(variable.Type()).(*types.Named).Obj().Name()
	switch {
	case !scalarAtomics[name]:
		return ", whose type " + diagnosticType(variable.Type()) + " is not atomic.Bool, Int32, Int64, Uint32, or Uint64"
	case variable.Pkg() != c.pass.Pkg:
		return " of another package, which any importer can change"
	case variable.Exported():
		return ", which other packages can change"
	case c.atomicCounters()[variable]:
		return ""
	}
	return ", which the package uses other than as the receiver of a direct method call"
}

// atomicCounters finds, once per package, the scalar atomic package variables whose every use is the receiver
// of a direct method call.
func (c *checks) atomicCounters() map[*types.Var]bool {
	if c.counters != nil {
		return c.counters
	}
	receivers := make(map[*ast.Ident]bool)
	for _, file := range c.pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				if selector, ok := call.Fun.(*ast.SelectorExpr); ok && c.pass.TypesInfo.Selections[selector] != nil {
					if id, ok := selector.X.(*ast.Ident); ok {
						receivers[id] = true
					}
				}
			}
			return true
		})
	}
	c.counters = make(map[*types.Var]bool)
	for id, object := range c.pass.TypesInfo.Uses {
		variable := packageVariable(object)
		if variable == nil || variable.Pkg() != c.pass.Pkg || !atomicType(variable.Type()) {
			continue
		}
		if valid, seen := c.counters[variable]; !seen || valid {
			c.counters[variable] = receivers[id]
		}
	}
	return c.counters
}
