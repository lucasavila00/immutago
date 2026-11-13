package borrow

import (
	"go/types"

	"golang.org/x/tools/go/ssa"
)

// normalizeProduced bounds local provenance by the finite compiler type graph.
// Re-entering a recursive type creates an opaque tail; subsequent projections absorb into it.
func (e *engine) normalizeProduced(point int, s state, instruction ssa.Instruction) {
	value, ok := instruction.(ssa.Value)
	if !ok {
		return
	}
	original := s.values[value]
	if len(original) == 0 {
		return
	}
	refs := make([]reference, 0, len(original))
	for _, r := range original {
		path, opaque := e.finitePath(r.place)
		r.place.Path = path
		if opaque {
			e.unsupportedRetention(r, instruction.Pos(), "unsupported recursive reference provenance")
		}
		refs, _ = mergeReferences(refs, []reference{r})
	}
	s.values[value] = refs
}

// finitePath identifies repeated reachable types without imposing an arbitrary recursion-depth limit.
func (e *engine) finitePath(place Place) ([]Projection, bool) {
	typ := e.roots[place.Root].typ
	seen := map[types.Type]bool{types.Unalias(typ): true}
	for index, projection := range place.Path {
		if projection.Kind == '*' {
			return place.Path[:index+1], true
		}
		next, known := projectionType(typ, projection)
		if !known {
			// Symbolic generic dereferences can be retained once; a repeated unknown edge is not a finite proof.
			for _, prior := range place.Path[:index] {
				if prior == projection {
					return opaquePath(place.Path, index), true
				}
			}
			continue
		}
		typ = types.Unalias(next)
		if seen[typ] {
			return opaquePath(place.Path, index), true
		}
		seen[typ] = true
	}
	return place.Path, false
}

// opaquePath replaces an unresolved recursive edge with an absorbing may-alias projection.
func opaquePath(path []Projection, index int) []Projection {
	result := append([]Projection(nil), path[:index]...)
	return append(result, Projection{Kind: '*'})
}

// projectionType follows Go field, element, and symbolic dereference types.
func projectionType(t types.Type, projection Projection) (types.Type, bool) {
	switch u := t.Underlying().(type) {
	case *types.Struct:
		if projection.Kind == 'f' && projection.Index >= 0 && projection.Index < int64(u.NumFields()) {
			return u.Field(int(projection.Index)).Type(), true
		}
	case *types.Array:
		if projection.Kind == 'i' || projection.Kind == '?' {
			return u.Elem(), true
		}
	case *types.Slice:
		if projection.Kind == 'i' || projection.Kind == '?' {
			return u.Elem(), true
		}
	case *types.Map:
		if projection.Kind == 'i' || projection.Kind == '?' {
			return u.Elem(), true
		}
	case *types.Chan:
		if projection.Kind == '?' {
			return u.Elem(), true
		}
	case *types.Pointer:
		if projection.Kind == 'd' {
			return u.Elem(), true
		}
	}
	return t, false
}
