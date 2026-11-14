package process

import "github.com/lucasavila00/immutago/pkg/proc"

// HiddenDependency removes the wrapper's named type identity.
type HiddenDependency proc.UncheckedDependency[*int]

// HiddenHolder stores a converted dependency in an unmarked field.
type HiddenHolder struct{ D HiddenDependency }

// hideDependency converts and stores a trusted value.
func hideDependency(p *int) HiddenHolder { return HiddenHolder{D: HiddenDependency(proc.Trust(p))} }

// EmbeddedDependency stores a dependency without a field name or marker.
type EmbeddedDependency struct{ proc.UncheckedDependency[*int] }

// PointerDependency stores a pointer to a dependency.
type PointerDependency struct {
	D *proc.UncheckedDependency[*int]
}

// ArrayDependency stores dependencies in an array.
type ArrayDependency struct {
	D [1]proc.UncheckedDependency[*int]
}

// boxDependency calls Value through an interface outside a process.
func boxDependency(d proc.UncheckedDependency[*int]) *int {
	var value interface{ Value() *int } = d
	return value.Value()
}

// variadicDependencies boxes a dependency in a variadic argument.
func variadicDependencies(d proc.UncheckedDependency[*int]) { consumeDependencies(d) }

// consumeDependencies accepts ordinary interface values.
func consumeDependencies(...any) {}

// pointerArguments creates a pointer to an argument-only struct.
func pointerArguments() { var args *SenderArgs; _ = args }

// arrayArguments creates an array of argument-only structs.
func arrayArguments() { var args [1]SenderArgs; _ = args }

// sliceDependencies creates a native collection of dependencies.
func sliceDependencies() { var deps []proc.UncheckedDependency[*int]; _ = deps }

// globalDependency stores a wrapper outside a local binding.
var globalDependency proc.UncheckedDependency[*int]

// globalArguments stores an argument-only struct in a package variable.
var globalArguments SenderArgs

// dependencyBuilder has valid ownership but an unsupported element type.
func dependencyBuilder(p *int) {
	values := []proc.UncheckedDependency[*int]{proc.Trust(p)}
	_ = len(values)
}

// argumentBuilder allocates native storage for an argument-only struct.
func argumentBuilder() { values := make([]SenderArgs, 1); _ = len(values) }

// dependencyKeyBuilder stores a dependency in a native map key.
func dependencyKeyBuilder() { values := map[proc.UncheckedDependency[*int]]int{}; _ = len(values) }
