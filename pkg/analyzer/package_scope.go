package analyzer

import (
	"go/ast"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
	"golang.org/x/tools/go/analysis"
)

// protectedPackage checks direct imports and locally used compiler type contracts.
func protectedPackage(pass *analysis.Pass) bool {
	return referencesLibrary(pass, true)
}

// referencesLibrary selects im operations, with an optional service result contract.
func referencesLibrary(pass *analysis.Pass, includeRuntime bool) bool {
	if importsCheckedLibrary(pass.Pkg, includeRuntime) {
		return true
	}
	seen := make(map[types.Type]bool)
	for _, value := range pass.TypesInfo.Types {
		if usesLibraryType(value.Type, seen, includeRuntime) {
			return true
		}
	}
	for _, objects := range []map[*ast.Ident]types.Object{pass.TypesInfo.Defs, pass.TypesInfo.Uses} {
		for _, object := range objects {
			if object != nil && usesLibraryType(object.Type(), seen, includeRuntime) {
				return true
			}
		}
	}
	for _, instance := range pass.TypesInfo.Instances {
		for i := 0; i < instance.TypeArgs.Len(); i++ {
			if usesLibraryType(instance.TypeArgs.At(i), seen, includeRuntime) {
				return true
			}
		}
	}
	return false
}

// usesLibraryType searches all local roots with one seen set; it does not cache per-root negative answers.
func usesLibraryType(t types.Type, seen map[types.Type]bool, includeRuntime bool) bool {
	if t == nil {
		return false
	}
	if seen[t] {
		return false
	}
	seen[t] = true
	switch value := t.(type) {
	case *types.Alias:
		return libraryTypeMetadata(value.Obj(), value.TypeParams(), value.TypeArgs(), seen, includeRuntime) ||
			usesLibraryType(value.Rhs(), seen, includeRuntime)
	case *types.Named:
		if checker.IsRuntimeType(value, "Ref") {
			return includeRuntime
		}
		return libraryTypeMetadata(value.Obj(), value.TypeParams(), value.TypeArgs(), seen, includeRuntime) ||
			usesLibraryType(value.Underlying(), seen, includeRuntime)
	case *types.TypeParam:
		return usesLibraryType(value.Constraint(), seen, includeRuntime)
	}
	return libraryTypeComponents(t, seen, includeRuntime)
}

// libraryTypeComponents follows the storage and callable edges of an unnamed compiler type.
func libraryTypeComponents(t types.Type, seen map[types.Type]bool, includeRuntime bool) bool {
	switch value := t.(type) {
	case *types.Pointer:
		return usesLibraryType(value.Elem(), seen, includeRuntime)
	case *types.Array:
		return usesLibraryType(value.Elem(), seen, includeRuntime)
	case *types.Slice:
		return usesLibraryType(value.Elem(), seen, includeRuntime)
	case *types.Map:
		return usesLibraryType(value.Key(), seen, includeRuntime) || usesLibraryType(value.Elem(), seen, includeRuntime)
	case *types.Chan:
		return usesLibraryType(value.Elem(), seen, includeRuntime)
	case *types.Struct:
		for field := range value.Fields() {
			if usesLibraryType(field.Type(), seen, includeRuntime) {
				return true
			}
		}
	case *types.Signature:
		return librarySignature(value, seen, includeRuntime)
	case *types.Tuple:
		return libraryTuple(value, seen, includeRuntime)
	case *types.Interface:
		return libraryInterface(value, seen, includeRuntime)
	case *types.Union:
		for i := 0; i < value.Len(); i++ {
			if usesLibraryType(value.Term(i).Type(), seen, includeRuntime) {
				return true
			}
		}
	}
	return false
}

// libraryTypeMetadata retains alias and phantom generic contracts before their underlying types erase them.
func libraryTypeMetadata(name *types.TypeName, parameters *types.TypeParamList,
	arguments *types.TypeList, seen map[types.Type]bool, includeRuntime bool) bool {
	if pkg := name.Pkg(); pkg != nil &&
		(pkg.Path() == checker.LibraryPath || (includeRuntime && pkg.Path() == checker.RuntimePath)) {
		return true
	}
	if libraryTypeParameters(parameters, seen, includeRuntime) {
		return true
	}
	for i := 0; i < arguments.Len(); i++ {
		if usesLibraryType(arguments.At(i), seen, includeRuntime) {
			return true
		}
	}
	return false
}

// libraryTuple checks each parameter or result with the same package-wide seen set.
func libraryTuple(value *types.Tuple, seen map[types.Type]bool, includeRuntime bool) bool {
	for variable := range value.Variables() {
		if usesLibraryType(variable.Type(), seen, includeRuntime) {
			return true
		}
	}
	return false
}

// librarySignature checks the used receiver, parameters, results, and generic constraints.
func librarySignature(signature *types.Signature, seen map[types.Type]bool, includeRuntime bool) bool {
	if receiver := signature.Recv(); receiver != nil && usesLibraryType(receiver.Type(), seen, includeRuntime) {
		return true
	}
	return usesLibraryType(signature.Params(), seen, includeRuntime) ||
		usesLibraryType(signature.Results(), seen, includeRuntime) ||
		libraryTypeParameters(signature.TypeParams(), seen, includeRuntime) ||
		libraryTypeParameters(signature.RecvTypeParams(), seen, includeRuntime)
}

// libraryTypeParameters checks constraints even when a generic parameter has no stored field.
func libraryTypeParameters(parameters *types.TypeParamList, seen map[types.Type]bool, includeRuntime bool) bool {
	for i := 0; i < parameters.Len(); i++ {
		if usesLibraryType(parameters.At(i).Constraint(), seen, includeRuntime) {
			return true
		}
	}
	return false
}

// libraryInterface follows explicit methods and embeddings without repeating inherited method sets.
func libraryInterface(value *types.Interface, seen map[types.Type]bool, includeRuntime bool) bool {
	for i := 0; i < value.NumExplicitMethods(); i++ {
		if usesLibraryType(value.ExplicitMethod(i).Type(), seen, includeRuntime) {
			return true
		}
	}
	for i := 0; i < value.NumEmbeddeds(); i++ {
		if usesLibraryType(value.EmbeddedType(i), seen, includeRuntime) {
			return true
		}
	}
	return false
}
