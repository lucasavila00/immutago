package borrow

import (
	"go/constant"
	"go/token"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
	"golang.org/x/tools/go/ssa"
)

// step interprets one compiler instruction without confusing administrative addresses with source loans.
func (e *engine) step(point int, s state) {
	instruction := e.instructions[point]
	switch op := instruction.(type) {
	case *ssa.Alloc:
		fresh := op.Comment == "complit" || op.Comment == "new" || op.Comment == "slicelit" || op.Comment == "makeslice"
		id := e.root(op, storageType(op.Type()), false, -1, fresh)
		s.values[op] = []reference{{place: Place{Root: id}, loan: -1}}
	case *ssa.MakeSlice:
		id := e.root(op, op.Type(), false, -1, true)
		s.values[op] = []reference{{place: Place{Root: id}, loan: -1}}
	case *ssa.FieldAddr:
		s.values[op] = e.project(point, op.Pos(), e.refs(s, op.X), 'f', int64(op.Field))
	case *ssa.IndexAddr:
		kind, index := indexProjection(op.Index)
		s.values[op] = e.project(point, op.Pos(), e.refs(s, op.X), kind, index)
	case *ssa.UnOp:
		e.load(point, s, op)
	case *ssa.Store:
		e.store(point, s, op)
	case *ssa.Slice:
		s.values[op] = e.sliceView(point, op.Pos(), e.refs(s, op.X))
	case *ssa.Phi:
		s.values[op] = e.phi(s, op)
	case *ssa.Lookup:
		s.values[op] = e.lookup(point, s, op)
	case *ssa.Call:
		s.values[op] = e.call(point, s, op.Common(), false, false)
	case *ssa.Defer:
		e.call(point, s, op.Common(), true, false)
	case *ssa.Go:
		e.call(point, s, op.Common(), false, true)
	case *ssa.MakeClosure:
		e.closure(point, s, op)
	case *ssa.Return:
		e.returned(point, s, op)
	case *ssa.DebugRef, *ssa.Jump: // Debug references are source metadata, not uses.
	default:
		if !e.convertValue(point, s, instruction) {
			e.other(point, s, instruction)
		}
	}
}

// project retains alias provenance while selecting a disjoint field or possibly overlapping index.
func (e *engine) project(point int, pos token.Pos, refs []reference, kind byte, index int64) []reference {
	result := make([]reference, 0, len(refs))
	for _, r := range refs {
		e.use(point, pos, r, false)
		r.place = r.place.project(kind, index)
		result = append(result, r)
	}
	return result
}

// indexProjection distinguishes statically disjoint array elements from unknown runtime indices.
func indexProjection(value ssa.Value) (byte, int64) {
	if c, ok := value.(*ssa.Const); ok && c.Value != nil && c.Value.Kind() == constant.Int {
		if index, ok := constant.Int64Val(c.Value); ok {
			return 'i', index
		}
	}
	return '?', 0
}

// copied records pointer-value uses without changing the origin of already shared references.
func (e *engine) copied(point int, pos token.Pos, refs []reference) []reference {
	for _, r := range refs {
		e.use(point, pos, r, false)
	}
	return refs
}

// load resolves pointer variables through memory and records ordinary pointee reads.
func (e *engine) load(point int, s state, op *ssa.UnOp) {
	refs := e.refs(s, op.X)
	for _, r := range refs {
		e.use(point, op.Pos(), r, false)
	}
	if op.Op == token.ARROW {
		var result []reference
		for _, r := range refs {
			r.shared = true
			r.place = r.place.project('?', 0)
			result = append(result, r)
		}
		s.values[op] = result
		return
	}
	if op.Op != token.MUL || !referenceType(op.Type()) {
		return
	}
	var result []reference
	for _, r := range refs {
		found := false
		for _, stored := range s.cells {
			if Overlap(r.place, stored.place, false) {
				result, _ = mergeReferences(result, stored.refs)
				found = true
			}
		}
		if !found && e.roots[r.place.Root].input >= 0 {
			r.place = r.place.project('d', 0)
			result = append(result, r)
		}
		if !found && e.roots[r.place.Root].global {
			id := e.root(op, storageType(op.Type()), true, -1, false)
			result = append(result, reference{place: Place{Root: id}, loan: -1, shared: true})
		}
	}
	s.values[op] = result
}

// store performs a strong update only for one definite local cell; may-alias writes use unions.
func (e *engine) store(point int, s state, op *ssa.Store) {
	addresses := e.refs(s, op.Addr)
	source := e.copied(point, op.Pos(), e.refs(s, op.Val))
	for _, address := range addresses {
		e.use(point, op.Pos(), address, true)
		values := e.storedReferences(point, op.Pos(), address, source)
		key := address.place.key()
		if len(addresses) == 1 && definitePlace(address.place) {
			s.cells[key] = cell{place: address.place, refs: values}
		} else {
			old := s.cells[key]
			joined, _ := mergeReferences(old.refs, values)
			s.cells[key] = cell{place: address.place, refs: joined}
		}
	}
}

// storedReferences gives aliases shared permission and records unsupported or externally retained origins.
func (e *engine) storedReferences(point int, pos token.Pos, address reference, source []reference) []reference {
	var values []reference
	for _, r := range source {
		origin := &e.roots[r.place.Root]
		if !r.shared && (origin.binding >= 0 && origin.binding != point || !origin.fresh) {
			r = e.borrow(r, point)
		}
		if !r.shared && origin.binding < 0 {
			origin.binding = point
		}
		values = append(values, r)
		if len(address.place.Path) > 0 {
			e.unsupportedRetention(r, pos, "unsupported borrowed reference stored in an aggregate")
		}
		target := e.roots[address.place.Root]
		if target.global && !target.protected && !checker.Contains(e.placeType(r.place)) {
			e.unsupportedRetention(r, pos, "unsupported borrowed reference stored in a global scalar slot")
		}
		if target.global {
			e.escape(point, r, false)
		}
	}
	return values
}

// definitePlace reports whether an address names one statically known storage cell.
func definitePlace(place Place) bool {
	for _, projection := range place.Path {
		if projection.Kind == '?' {
			return false
		}
	}
	return true
}

// other records remaining reads and rejects unsupported escapes of protected references.
func (e *engine) other(point int, s state, instruction ssa.Instruction) {
	var refs []reference
	for _, operand := range instruction.Operands(nil) {
		if operand != nil && *operand != nil {
			refs, _ = mergeReferences(refs, e.refs(s, *operand))
		}
	}
	for _, r := range refs {
		e.use(point, instruction.Pos(), r, false)
	}
	switch instruction.(type) {
	case *ssa.Send, *ssa.MapUpdate, *ssa.Panic, *ssa.Select:
		for _, r := range refs {
			e.unsupportedRetention(r, instruction.Pos(), "unsupported reference escape through mutable container or channel")
		}
	}
}

// globallyProtected distinguishes immutable global values from rebindable pointer slots.
func globallyProtected(t types.Type) bool {
	switch t.Underlying().(type) {
	case *types.Pointer, *types.Slice, *types.Map, *types.Interface, *types.Signature:
		return false
	}
	return true
}

// unsupportedRetention preserves unproved effects through scalar and generic helper summaries.
func (e *engine) unsupportedRetention(r reference, pos token.Pos, message string) {
	if !e.record {
		return
	}
	origin := e.roots[r.place.Root]
	if origin.input >= 0 {
		e.summary.Inputs[origin.input].Unknown = true
	}
	if origin.protected {
		e.report(pos, message)
	}
}

// sliceView preserves fresh owned backing arrays and borrows existing arrays.
func (e *engine) sliceView(point int, pos token.Pos, refs []reference) []reference {
	out := make([]reference, 0, len(refs))
	for _, r := range refs {
		e.use(point, pos, r, false)
		if !e.roots[r.place.Root].fresh || r.shared {
			r = e.borrow(r, point)
		}
		out = append(out, r)
	}
	return out
}

// phi joins reference origins from SSA control-flow alternatives.
func (e *engine) phi(s state, op *ssa.Phi) []reference {
	var refs []reference
	for _, edge := range op.Edges {
		refs, _ = mergeReferences(refs, e.refs(s, edge))
	}
	return refs
}

// asserted refines unknown reference results when a type assertion exposes protected storage.
func (e *engine) asserted(point int, s state, op *ssa.TypeAssert) []reference {
	refs := e.copied(point, op.Pos(), e.refs(s, op.X))
	for _, r := range refs {
		if checker.Contains(storageType(op.AssertedType)) ||
			e.roots[r.place.Root].unknown && potentialInterior(storageType(op.AssertedType)) {
			e.roots[r.place.Root].protected = true
		}
	}
	return refs
}

// lookup keeps native map reference elements shared and tied to the map input.
func (e *engine) lookup(point int, s state, op *ssa.Lookup) []reference {
	var refs []reference
	for _, r := range e.refs(s, op.X) {
		e.use(point, op.Pos(), r, false)
		r.shared = true
		r.place = r.place.project('?', 0)
		refs = append(refs, r)
	}
	return refs
}

// extract selects one result while retaining the reference's root and permission.
func extract(refs []reference, index int) []reference {
	var result []reference
	for _, r := range refs {
		if r.result == index {
			r.result = 0
			result = append(result, r)
		}
	}
	return result
}

// placeType recovers the static type of an interior reference for checked global publication.
func (e *engine) placeType(place Place) types.Type {
	typ := e.roots[place.Root].typ
	for _, projection := range place.Path {
		switch underlying := typ.Underlying().(type) {
		case *types.Struct:
			if projection.Kind != 'f' || projection.Index < 0 || projection.Index >= int64(underlying.NumFields()) {
				return typ
			}
			typ = underlying.Field(int(projection.Index)).Type()
		case *types.Array:
			typ = underlying.Elem()
		case *types.Slice:
			typ = underlying.Elem()
		case *types.Map:
			typ = underlying.Elem()
		case *types.Pointer:
			typ = underlying.Elem()
		default:
			return typ
		}
	}
	return typ
}

// convertValue preserves references across type erasure and tuple extraction.
func (e *engine) convertValue(point int, s state, instruction ssa.Instruction) bool {
	switch op := instruction.(type) {
	case *ssa.ChangeType:
		s.values[op] = e.copied(point, op.Pos(), e.refs(s, op.X))
	case *ssa.Convert:
		s.values[op] = e.copied(point, op.Pos(), e.refs(s, op.X))
	case *ssa.ChangeInterface:
		s.values[op] = e.copied(point, op.Pos(), e.refs(s, op.X))
	case *ssa.MakeInterface:
		s.values[op] = e.copied(point, op.Pos(), e.refs(s, op.X))
	case *ssa.TypeAssert:
		s.values[op] = e.asserted(point, s, op)
	case *ssa.Range:
		s.values[op] = e.copied(point, op.Pos(), e.refs(s, op.X))
	case *ssa.Next:
		s.values[op] = e.tupleReferences(point, op.Pos(), e.refs(s, op.Iter), op.Type())
	case *ssa.Index:
		kind, index := indexProjection(op.Index)
		refs := e.project(point, op.Pos(), e.refs(s, op.X), kind, index)
		for i := range refs {
			refs[i].shared = true
		}
		s.values[op] = refs
	case *ssa.Select:
		e.other(point, s, op)
		var refs []reference
		for _, selected := range op.States {
			refs, _ = mergeReferences(refs, e.refs(s, selected.Chan))
		}
		s.values[op] = e.tupleReferences(point, op.Pos(), refs, op.Type())
	case *ssa.Extract:
		s.values[op] = extract(e.refs(s, op.Tuple), op.Index)
	default:
		return false
	}
	return true
}

// tupleReferences preserves element aliases returned by map iterators and channel selection.
func (e *engine) tupleReferences(point int, pos token.Pos, refs []reference, typ types.Type) []reference {
	tuple, ok := typ.(*types.Tuple)
	if !ok {
		return nil
	}
	var result []reference
	for index := 0; index < tuple.Len(); index++ {
		if !referenceType(tuple.At(index).Type()) {
			continue
		}
		for _, r := range refs {
			e.use(point, pos, r, false)
			r.shared = true
			r.result = index
			r.place = r.place.project('?', 0)
			result = append(result, r)
		}
	}
	return result
}
