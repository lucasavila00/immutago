package analyzer

import (
	"go/token"
	"go/types"
)

// checkTrustedPackage requires a named type argument that the current package declares.
func (c *checks) checkTrustedPackage(pos token.Pos, argument types.Type, use string) {
	if c.queries.Views(argument) {
		c.report(pos, "unchecked storage cannot contain local frozen views")
		return
	}
	named, ok := types.Unalias(argument).(*types.Named)
	if ok && named.Obj().Pkg() == c.pass.Pkg {
		return
	}
	c.report(pos, use+" requires a type that this package declares\n  type: "+diagnosticType(argument)+
		"\n  rule: only the package that declares T can wrap or read an im.UncheckedValue[T]"+
		"\n  fix: call a function of the package that declares the type")
}
