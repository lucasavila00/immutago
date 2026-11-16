// Package checker proves structural immutability from Go compiler type information.
// It follows Go's structural type predicates and checks each component.
// Frozen type obligations form a graph; definite failures propagate to owners.
// A recursive collection is accepted only when every reachable obligation holds.
package checker

import (
	"go/types"
)

// LibraryPath is the package whose sealed collection implementations are trusted.
const LibraryPath = "github.com/lucasavila00/immutago/pkg/im"

// RuntimePath is the package whose supervised service runtime is trusted.
const RuntimePath = "github.com/lucasavila00/immutago/pkg/supervise"

// ServiceMarker identifies supervise.Service, the generic type embedded by services.
const ServiceMarker = "Service"

// IsLibraryType identifies a library type by package identity and name, not spelling at its use site.
func IsLibraryType(t types.Type, name string) bool {
	n, ok := types.Unalias(t).(*types.Named)
	return ok && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == LibraryPath && n.Obj().Name() == name
}

// IsRuntimeType identifies a service runtime type by package identity and name, not spelling at its use site.
func IsRuntimeType(t types.Type, name string) bool {
	n, ok := types.Unalias(t).(*types.Named)
	return ok && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == RuntimePath && n.Obj().Name() == name
}

// IsReference identifies Ref and defined types that retain its sealed representation.
// Export data can omit the Ref declaration but keeps the private field's package identity.
func IsReference(t types.Type) bool {
	if IsRuntimeType(t, "Ref") {
		return true
	}
	structure, ok := t.Underlying().(*types.Struct)
	if !ok || structure.NumFields() != 1 {
		return false
	}
	field := structure.Field(0)
	if field.Pkg() == nil || field.Pkg().Path() != RuntimePath || field.Exported() ||
		field.Embedded() || structure.Tag(0) != "" {
		return false
	}
	pointer, ok := types.Unalias(field.Type()).(*types.Pointer)
	if !ok {
		return false
	}
	slot, ok := types.Unalias(pointer.Elem()).(*types.Named)
	return ok && slot.Obj().Pkg() != nil && slot.Obj().Pkg().Path() == "sync/atomic" &&
		slot.Obj().Name() == "Pointer" && slot.TypeArgs().Len() == 1
}

// IsUncheckedValue reports whether t is the library's unchecked immutable value wrapper.
func IsUncheckedValue(t types.Type) bool { return IsLibraryType(t, "UncheckedValue") }

// IsCollection reports whether t is one of the library's sealed collection types.
func IsCollection(t types.Type) bool {
	return IsLibraryType(t, "FrozenSlice") || IsLibraryType(t, "FrozenMap") || IsLibraryType(t, "FrozenString")
}

// IsProjection identifies the fixed local read types.
func IsProjection(t types.Type) bool {
	return IsLibraryType(t, "Projection") || IsLibraryType(t, "StringProjection") ||
		IsLibraryType(t, "SliceProjection") || IsLibraryType(t, "MapProjection")
}

// Marked reports an embedded Frozen or Local marker, including invalid combinations.
// It does not prove immutability; Reason checks the complete type contract.
func Marked(t types.Type) bool {
	return Frozen(t) || Local(t)
}

// Frozen reports the presence of an embedded im.Frozen value, not type validity.
func Frozen(t types.Type) bool { return embedsMarker(t, "Frozen") }

// Local reports the presence of an embedded im.Local value, not type validity.
func Local(t types.Type) bool { return embedsMarker(t, "Local") }

// Copied reports the presence of an embedded im.Copied value, not type validity.
func Copied(t types.Type) bool { return embedsMarker(t, "Copied") }

// Service reports whether a struct embeds supervise.Service, directly or indirectly.
func Service(t types.Type) bool {
	seen := make(map[types.Type]bool)
	pending := []types.Type{t}
	for len(pending) != 0 {
		current := types.Unalias(pending[len(pending)-1])
		pending = pending[:len(pending)-1]
		if pointer, ok := current.(*types.Pointer); ok {
			current = types.Unalias(pointer.Elem())
		}
		if seen[current] {
			continue
		}
		seen[current] = true
		if IsRuntimeType(current, ServiceMarker) {
			return true
		}
		if structure, ok := current.Underlying().(*types.Struct); ok {
			for field := range structure.Fields() {
				if field.Embedded() {
					pending = append(pending, field.Type())
				}
			}
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
		if field.Embedded() && (IsLibraryType(field.Type(), "Frozen") || IsLibraryType(field.Type(), "Local") ||
			IsLibraryType(field.Type(), "Copied")) {
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

// Contains reports values that need im source checks, including their holders and containers.
// It runs one query; use Classifier to reuse results across queries.
func Contains(t types.Type) bool {
	var classifier Classifier
	return classifier.Contains(t)
}
