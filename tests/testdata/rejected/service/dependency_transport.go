package service

import "github.com/lucasavila00/immutago/pkg/supervise"

// HiddenDependency removes the wrapper's named type identity.
type HiddenDependency supervise.ConcurrentSafe[*int]

// HiddenHolder stores a converted dependency in an unmarked field.
type HiddenHolder struct{ D HiddenDependency }

// hideDependency converts and stores a trusted value.
func hideDependency(p *int) HiddenHolder {
	return HiddenHolder{D: HiddenDependency(supervise.AssumeConcurrentSafe(p))}
}

// EmbeddedDependency stores a dependency without a field name or marker.
type EmbeddedDependency struct{ supervise.ConcurrentSafe[*int] }

// PointerDependency stores a pointer to a dependency.
type PointerDependency struct {
	D *supervise.ConcurrentSafe[*int]
}

// ArrayDependency stores dependencies in an array.
type ArrayDependency struct {
	D [1]supervise.ConcurrentSafe[*int]
}

// boxDependency calls Value through an interface outside a service.
func boxDependency(d supervise.ConcurrentSafe[*int]) *int {
	var value interface{ Value() *int } = d
	return value.Value()
}

// variadicDependencies boxes a dependency in a variadic argument.
func variadicDependencies(d supervise.ConcurrentSafe[*int]) { consumeDependencies(d) }

// consumeDependencies accepts ordinary interface values.
func consumeDependencies(...any) {}

// pointerArguments creates a pointer to an argument-only struct.
func pointerArguments() { var args *SenderArgs; _ = args }

// arrayArguments creates an array of argument-only structs.
func arrayArguments() { var args [1]SenderArgs; _ = args }

// sliceDependencies creates a native collection of dependencies.
func sliceDependencies() { var deps []supervise.ConcurrentSafe[*int]; _ = deps }

// globalDependency stores a wrapper outside a local binding.
var globalDependency supervise.ConcurrentSafe[*int]

// globalArguments stores an argument-only struct in a package variable.
var globalArguments SenderArgs

// dependencyBuilder has valid ownership but an unsupported element type.
func dependencyBuilder(p *int) {
	values := []supervise.ConcurrentSafe[*int]{supervise.AssumeConcurrentSafe(p)}
	_ = len(values)
}

// argumentBuilder allocates native storage for an argument-only struct.
func argumentBuilder() { values := make([]SenderArgs, 1); _ = len(values) }

// dependencyKeyBuilder stores a dependency in a native map key.
func dependencyKeyBuilder() { values := map[supervise.ConcurrentSafe[*int]]int{}; _ = len(values) }
