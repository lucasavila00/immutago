// Package checker proves structural immutability from Go compiler type information.
// It follows Go's recursive type predicates: inspect the underlying type,
// check each component, and stop revisiting types on the current proof path.
// A recursive collection is accepted only when every nonrecursive edge is safe.
package checker

import (
	"fmt"
	"go/types"
)

// LibraryPath is the package whose sealed collection implementations are trusted.
const LibraryPath = "github.com/lucasavila00/immutago/pkg/immutable"

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

// Reason returns an empty string when t is proved immutable, or the first failed obligation.
func Reason(t types.Type) string { return prove(t, make(map[types.Type]bool)) }

// prove checks all component obligations, with a path-local cycle guard.
func prove(t types.Type, active map[types.Type]bool) string {
	t = types.Unalias(t)
	if active[t] {
		return ""
	}
	active[t] = true
	defer delete(active, t)
	if IsLibraryType(t, "Marker") {
		return ""
	}
	if IsCollection(t) {
		args := t.(*types.Named).TypeArgs()
		for i := 0; i < args.Len(); i++ {
			if reason := prove(args.At(i), active); reason != "" {
				return fmt.Sprintf("type argument %d: %s", i+1, reason)
			}
		}
		return ""
	}
	if p, ok := t.(*types.TypeParam); ok {
		return proveConstraint(p.Constraint(), active)
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		if u.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat|types.IsComplex|types.IsString) != 0 {
			return ""
		}
	case *types.Array:
		return prove(u.Elem(), active)
	case *types.Struct:
		if !Marked(t) {
			return "struct does not embed immutable.Marker"
		}
		for field := range u.Fields() {
			if reason := prove(field.Type(), active); reason != "" {
				return field.Name() + ": " + reason
			}
		}
		return ""
	}
	return fmt.Sprintf("%s is not provably immutable", types.TypeString(t, nil))
}

// proveConstraint accepts a type parameter when an embedded type set proves all possible members safe.
func proveConstraint(t types.Type, active map[types.Type]bool) string {
	iface, ok := t.Underlying().(*types.Interface)
	if !ok {
		return prove(t, active)
	}
	for i := 0; i < iface.NumEmbeddeds(); i++ {
		embedded := iface.EmbeddedType(i)
		if union, ok := embedded.(*types.Union); ok {
			safe := true
			for j := 0; j < union.Len(); j++ {
				if prove(union.Term(j).Type(), active) != "" {
					safe = false
					break
				}
			}
			if safe {
				return ""
			}
		} else if proveConstraint(embedded, active) == "" {
			return ""
		}
	}
	return "type parameter constraint does not prove immutability"
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
