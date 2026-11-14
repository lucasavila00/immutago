package analyzer

import (
	"go/ast"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// hasHolderReference finds holder or slot pointers within a type.
func (c *subsetChecks) hasHolderReference(t types.Type) bool {
	if c.holderRefs == nil {
		c.holderRefs = make(map[types.Type]bool)
	}
	if result, ok := c.holderRefs[t]; ok {
		return result
	}
	seen := make(map[types.Type]bool)
	result := c.findHolderReference(t, seen)
	if !result {
		for visited := range seen {
			c.holderRefs[visited] = false
		}
	}
	c.holderRefs[t] = result
	return result
}

// findHolderReference walks storage while keeping the sealed slot opaque.
func (c *subsetChecks) findHolderReference(t types.Type, seen map[types.Type]bool) bool {
	if t == nil || seen[t] || checker.IsLibraryType(t, "Owner") {
		return false
	}
	if result, ok := c.holderRefs[t]; ok {
		return result
	}
	seen[t] = true
	switch value := t.Underlying().(type) {
	case *types.Pointer:
		return c.holderPointer(t) || checker.IsLibraryType(value.Elem(), "Owner") ||
			c.findHolderReference(value.Elem(), seen)
	case *types.Array:
		return c.findHolderReference(value.Elem(), seen)
	case *types.Slice:
		return c.findHolderReference(value.Elem(), seen)
	case *types.Map:
		return c.findHolderReference(value.Key(), seen) || c.findHolderReference(value.Elem(), seen)
	case *types.Chan:
		return c.findHolderReference(value.Elem(), seen)
	case *types.Struct:
		return c.structHolderReference(value, seen)
	case *types.Interface:
		return c.interfaceHolderReference(value, seen)
	case *types.Union:
		return c.unionHolderReference(value, seen)
	case *types.Signature:
		return c.findHolderReference(value.Params(), seen) || c.findHolderReference(value.Results(), seen)
	case *types.Tuple:
		return c.tupleHolderReference(value, seen)
	}
	return false
}

// unionHolderReference checks each term in a type set.
func (c *subsetChecks) unionHolderReference(value *types.Union, seen map[types.Type]bool) bool {
	for i := 0; i < value.Len(); i++ {
		if c.findHolderReference(value.Term(i).Type(), seen) {
			return true
		}
	}
	return false
}

// structHolderReference checks fields for stored holder pointers.
func (c *subsetChecks) structHolderReference(value *types.Struct, seen map[types.Type]bool) bool {
	for field := range value.Fields() {
		if c.findHolderReference(field.Type(), seen) {
			return true
		}
	}
	return false
}

// tupleHolderReference checks parameter and result types.
func (c *subsetChecks) tupleHolderReference(value *types.Tuple, seen map[types.Type]bool) bool {
	for i := 0; i < value.Len(); i++ {
		if c.findHolderReference(value.At(i).Type(), seen) {
			return true
		}
	}
	return false
}

// interfaceHolderReference checks the complete interface method set.
func (c *subsetChecks) interfaceHolderReference(value *types.Interface, seen map[types.Type]bool) bool {
	for i := 0; i < value.NumMethods(); i++ {
		if c.findHolderReference(value.Method(i).Type(), seen) {
			return true
		}
	}
	for i := 0; i < value.NumEmbeddeds(); i++ {
		if c.findHolderReference(value.EmbeddedType(i), seen) {
			return true
		}
	}
	return false
}

// checkHolderSignature permits only direct, fixed holder pointer parameters.
func (c *subsetChecks) checkHolderSignature(sig *types.Signature, direct bool, pos ast.Node) {
	if c.hasHolderReference(sig.Results()) {
		c.pass.Reportf(pos.Pos(), "Owner holder pointers cannot be returned")
	}
	for i := 0; i < sig.Params().Len(); i++ {
		parameter := sig.Params().At(i).Type()
		if !c.hasHolderReference(parameter) {
			continue
		}
		if direct && c.holderPointer(parameter) && (!sig.Variadic() || i != sig.Params().Len()-1) {
			continue
		}
		c.pass.Reportf(pos.Pos(), "Owner holder pointers require direct non-variadic parameters")
	}
}
