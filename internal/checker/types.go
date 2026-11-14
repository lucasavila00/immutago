// Package checker proves structural immutability from Go compiler type information.
// It follows Go's structural type predicates and checks each component.
// Shared type obligations form a graph; definite failures propagate to owners.
// A recursive collection is accepted only when every reachable obligation holds.
package checker

import (
	"go/types"
)

// LibraryPath is the package whose sealed collection implementations are trusted.
const LibraryPath = "github.com/lucasavila00/immutago/pkg/im"

// RuntimePath is the package whose supervised process runtime is trusted.
const RuntimePath = "github.com/lucasavila00/immutago/pkg/proc"

// Names of the process runtime identities in RuntimePath and of the process method that only the runtime calls.
const (
	ProcessMarker         = "Process"
	PIDType               = "PID"
	StartFunction         = "Start"
	ReserveFunction       = "Reserve"
	StartReservedFunction = "StartReserved"
	ProcMethod            = "Proc"
	ArgsMethod            = "Args"
	RunMethod             = "Run"
)

// IsLibraryType identifies a library type by package identity and name, not spelling at its use site.
func IsLibraryType(t types.Type, name string) bool {
	n, ok := types.Unalias(t).(*types.Named)
	return ok && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == LibraryPath && n.Obj().Name() == name
}

// IsRuntimeType identifies a process runtime type by package identity and name, not spelling at its use site.
func IsRuntimeType(t types.Type, name string) bool {
	n, ok := types.Unalias(t).(*types.Named)
	return ok && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == RuntimePath && n.Obj().Name() == name
}

// IsUnchecked reports whether t is the library's trusted opaque wrapper.
func IsUnchecked(t types.Type) bool { return IsLibraryType(t, "Unchecked") }

// IsCollection reports whether t is one of the library's sealed collection types.
func IsCollection(t types.Type) bool {
	return IsLibraryType(t, "FrozenSlice") || IsLibraryType(t, "FrozenMap")
}

// Marked reports the presence of either embedded marker, including invalid combinations.
// It does not prove immutability; Reason checks the complete type contract.
func Marked(t types.Type) bool {
	return Shared(t) || Scoped(t)
}

// Shared reports the presence of an embedded im.Shared value, not type validity.
func Shared(t types.Type) bool { return embedsMarker(t, "Shared") }

// Scoped reports the presence of an embedded im.Scoped value, not type validity.
func Scoped(t types.Type) bool { return embedsMarker(t, "Scoped") }

// Process reports the presence of an embedded proc.Process value. A process is not a marked struct.
func Process(t types.Type) bool {
	if t == nil {
		return false
	}
	s, ok := t.Underlying().(*types.Struct)
	if !ok {
		return false
	}
	for field := range s.Fields() {
		if field.Embedded() && IsRuntimeType(field.Type(), ProcessMarker) {
			return true
		}
	}
	return false
}

// MarkerCount counts direct embedded marker identities, including aliases of the same marker.
func MarkerCount(t types.Type) int {
	if t == nil {
		return 0
	}
	structure, ok := t.Underlying().(*types.Struct)
	if !ok {
		return 0
	}
	count := 0
	for field := range structure.Fields() {
		if field.Embedded() && (IsLibraryType(field.Type(), "Shared") || IsLibraryType(field.Type(), "Scoped")) {
			count++
		}
	}
	return count
}

// embedsMarker resolves one marker through the compiler's embedded-field identities.
func embedsMarker(t types.Type, name string) bool {
	if t == nil {
		return false
	}
	s, ok := t.Underlying().(*types.Struct)
	if !ok {
		return false
	}
	for field := range s.Fields() {
		if field.Embedded() && IsLibraryType(field.Type(), name) {
			return true
		}
	}
	return false
}

// Contains reports whether a type stores a marked value or sealed collection.
// It runs one query; use Classifier to reuse results across queries.
func Contains(t types.Type) bool {
	var classifier Classifier
	return classifier.Contains(t)
}
