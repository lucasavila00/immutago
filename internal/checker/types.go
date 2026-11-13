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

// IsLibraryType identifies a library type by package identity and name, not spelling at its use site.
func IsLibraryType(t types.Type, name string) bool {
	n, ok := types.Unalias(t).(*types.Named)
	return ok && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == LibraryPath && n.Obj().Name() == name
}

// IsCollection reports whether t is one of the library's sealed collection types.
func IsCollection(t types.Type) bool {
	return IsLibraryType(t, "Slice") || IsLibraryType(t, "Map")
}

// Marked reports whether a struct directly embeds Marker by value.
func Marked(t types.Type) bool {
	if t == nil {
		return false
	}
	s, ok := t.Underlying().(*types.Struct)
	if !ok {
		return false
	}
	for field := range s.Fields() {
		if field.Embedded() && IsLibraryType(field.Type(), "Marker") {
			return true
		}
	}
	return false
}

// Contains reports whether a type stores a marked value or sealed collection.
// It traverses containers so an unmarked wrapper cannot hide a marked field.
func Contains(t types.Type) bool { return contains(t, make(map[types.Type]bool)) }

// contains walks storage types and terminates on recursive declarations.
func contains(t types.Type, seen map[types.Type]bool) bool {
	if t == nil || seen[t] {
		return false
	}
	seen[t] = true
	if parameter, ok := types.Unalias(t).(*types.TypeParam); ok {
		return constraintContains(parameter.Constraint(), seen)
	}
	if IsCollection(t) || Marked(t) {
		return true
	}
	switch u := t.Underlying().(type) {
	case *types.Pointer:
		return contains(u.Elem(), seen)
	case *types.Array:
		return contains(u.Elem(), seen)
	case *types.Slice:
		return contains(u.Elem(), seen)
	case *types.Map:
		return contains(u.Key(), seen) || contains(u.Elem(), seen)
	case *types.Chan:
		return contains(u.Elem(), seen)
	case *types.Struct:
		for field := range u.Fields() {
			if contains(field.Type(), seen) {
				return true
			}
		}
	}
	return false
}

// constraintContains finds immutable storage explicitly named by a generic type set.
func constraintContains(t types.Type, seen map[types.Type]bool) bool {
	switch u := t.Underlying().(type) {
	case *types.Interface:
		for i := 0; i < u.NumEmbeddeds(); i++ {
			if constraintContains(u.EmbeddedType(i), seen) {
				return true
			}
		}
	case *types.Union:
		for i := 0; i < u.Len(); i++ {
			if contains(u.Term(i).Type(), seen) {
				return true
			}
		}
	default:
		return contains(t, seen)
	}
	return false
}
