package checker

import (
	"fmt"
	"go/types"
)

// FrozenInterface identifies the sealed method on an interface by compiler identity.
// It does not prove the interface's name or method contracts.
func FrozenInterface(t types.Type) bool {
	if t == nil {
		return false
	}
	contract, ok := t.Underlying().(*types.Interface)
	if !ok {
		return false
	}
	for method := range contract.Methods() {
		if FrozenInterfaceSeal(method) {
			return true
		}
	}
	return false
}

// FrozenInterfaceSeal identifies the private method supplied by im.Frozen.
func FrozenInterfaceSeal(method *types.Func) bool {
	return method.Pkg() != nil && method.Pkg().Path() == LibraryPath && method.Name() == "immutagoFrozen"
}

// frozenInterface proves a named contract whose methods return only read results.
func (p *proof) frozenInterface(result *obligation, t types.Type) {
	if !FrozenInterface(t) {
		result.reason = fmt.Sprintf("%s is not provably immutable", types.TypeString(t, nil))
		return
	}
	named, ok := t.(*types.Named)
	if !ok || named.TypeParams().Len() != 0 {
		result.reason = "Frozen interfaces require a named non-generic interface"
		return
	}
	if IsLibraryType(t, "FrozenInterface") {
		result.reason = "immutable markers are annotations, not data"
		return
	}
	contract := named.Underlying().(*types.Interface)
	if !contract.IsMethodSet() {
		result.reason = "Frozen interfaces require a method set without type terms"
		return
	}
	for method := range contract.Methods() {
		if FrozenInterfaceSeal(method) {
			continue
		}
		if !method.Exported() {
			result.reason = "Frozen interface methods must be exported"
			return
		}
		signature := method.Type().(*types.Signature)
		if signature.Variadic() || signature.Results().Len() > 1 {
			result.reason = "Frozen interface methods require fixed parameters and at most one read result"
			return
		}
		for parameter := range signature.Params().Variables() {
			switch parameter.Type().Underlying().(type) {
			case *types.Interface, *types.Signature:
				result.reason = "Frozen interface parameters require concrete types"
				return
			}
		}
		if signature.Results().Len() == 1 {
			p.dependCategory(result, signature.Results().At(0).Type(), fmt.Sprintf("%s result: ", method.Name()), readCategory)
		}
	}
}
