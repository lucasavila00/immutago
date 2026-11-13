package borrow

import (
	"fmt"
	"go/token"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
	"golang.org/x/tools/go/ssa"
)

// call applies checked input effects and carries returned aliases back to the caller.
func (e *engine) call(point int, s state, call *ssa.CallCommon, deferred, asynchronous bool) []reference {
	if builtin, ok := call.Value.(*ssa.Builtin); ok {
		return e.builtin(point, s, call, builtin.Name())
	}
	arguments := make([][]reference, len(call.Args))
	for i, arg := range call.Args {
		arguments[i] = append([]reference(nil), e.refs(s, arg)...)
	}
	callee := e.callee(point, s, call)
	arguments = e.closureArguments(point, s, call, callee, arguments)
	if isLibrary(callee) {
		for _, args := range arguments {
			e.copied(point, call.Pos(), args)
		}
		return e.libraryResults(point, call, callee)
	}
	summary, known := e.program.summary(callee)
	if !known {
		e.unknownCall(point, call.Pos(), append(arguments, e.refs(s, call.Value)))
		return e.unknownResults(point, call)
	}
	e.argumentEffects(point, s, call, summary, arguments, deferred, asynchronous)
	return e.callResults(point, s, call, summary, arguments)
}

// callee resolves direct and locally tracked closure calls without assuming dynamic dispatch is pure.
func (e *engine) callee(point int, s state, call *ssa.CallCommon) *ssa.Function {
	if fn := call.StaticCallee(); fn != nil {
		return fn
	}
	var callee *ssa.Function
	for _, r := range e.refs(s, call.Value) {
		e.use(point, call.Pos(), r, false)
		if r.closure == nil {
			continue
		}
		if callee != nil && callee != r.closure {
			return nil
		}
		callee = r.closure
	}
	return callee
}

// closureArguments connects captured variables to the closure body's symbolic free-variable inputs.
func (e *engine) closureArguments(point int, s state, call *ssa.CallCommon, callee *ssa.Function,
	arguments [][]reference) [][]reference {
	if callee == nil || len(callee.FreeVars) == 0 {
		return arguments
	}
	for range callee.FreeVars {
		arguments = append(arguments, nil)
	}
	for _, r := range e.refs(s, call.Value) {
		if r.binding < 0 {
			continue
		}
		index := len(call.Args) + r.binding
		r.closure = nil
		r.binding = 0
		arguments[index] = append(arguments[index], r)
		e.reachableUses(point, call.Pos(), s, r, false)
	}
	return arguments
}

// argumentEffects applies a summary to each argument without mutating saved provenance snapshots.
func (e *engine) argumentEffects(point int, s state, call *ssa.CallCommon, summary Summary,
	arguments [][]reference, deferred, asynchronous bool) {
	for i, args := range arguments {
		effect := Effect{}
		if i < len(summary.Inputs) {
			effect = summary.Inputs[i]
		}
		for j, r := range args {
			r = e.borrow(r, point)
			arguments[i][j] = r
			e.argumentEffect(point, s, call.Pos(), r, effect, deferred, asynchronous)
		}
	}
}

// argumentEffect checks one shared input and propagates unknown or retained effects to callers.
func (e *engine) argumentEffect(point int, s state, pos token.Pos, r reference,
	effect Effect, deferred, asynchronous bool) {
	e.use(point, pos, r, effect.Write)
	e.reachableUses(point, pos, s, r, effect.Write)
	if effect.Unknown {
		e.unsupportedRetention(r, pos, "unsupported call effects for borrowed immutable storage")
	}
	if effect.Capture || asynchronous {
		e.escape(point, r, false)
	}
	if deferred {
		e.escape(point, r, true)
	}
}

// isLibrary recognizes only the fixed collection implementation boundary.
func isLibrary(fn *ssa.Function) bool {
	return fn != nil && fn.Pkg != nil && fn.Pkg.Pkg.Path() == checker.LibraryPath
}

// unknownCall rejects unavailable effects only when protected references cross the call boundary.
func (e *engine) unknownCall(point int, pos token.Pos, arguments [][]reference) {
	for _, args := range arguments {
		for _, r := range args {
			e.use(point, pos, r, false)
			if e.record && e.roots[r.place.Root].input >= 0 {
				e.summary.Inputs[e.roots[r.place.Root].input].Unknown = true
			}
			if e.roots[r.place.Root].protected {
				e.report(pos, "unsupported call effects for borrowed immutable storage")
			}
		}
	}
}

// callResults instantiates symbolic returned aliases and keeps repeated fresh results on one root.
func (e *engine) callResults(point int, s state, call *ssa.CallCommon,
	summary Summary, args [][]reference) []reference {
	var result []reference
	freshSeen := make(map[string]bool)
	for index, aliases := range summary.Results {
		for _, alias := range aliases {
			if alias.Input >= 0 && alias.Input < len(args) {
				result = append(result, e.inputResults(s, args[alias.Input], alias.Path, index)...)
			} else if alias.Fresh || alias.Global || alias.Unknown {
				result = append(result, e.freshResult(point, call, alias, index, freshSeen))
			}
		}
	}
	return result
}

// inputResults substitutes a returned parameter path into each possible caller origin.
func (e *engine) inputResults(s state, args []reference, path []Projection, index int) []reference {
	var result []reference
	for _, base := range args {
		for _, r := range e.resolvePath(s, base, path) {
			r.result = index
			result = append(result, r)
		}
	}
	return result
}

// freshResult creates one caller allocation per returned alias group and preserves shared output permissions.
func (e *engine) freshResult(point int, call *ssa.CallCommon, alias Result, index int, seen map[string]bool) reference {
	key := fmt.Sprintf("%d:%s:%t", point, alias.FreshGroup, alias.Global)
	id, ok := e.callRoots[key]
	if !ok {
		typ := storageType(call.Signature().Results().At(index).Type())
		id = len(e.roots)
		e.callRoots[key] = id
		e.roots = append(e.roots, root{typ: typ, protected: alias.Protected || checker.Contains(typ),
			input: -1, global: alias.Global, synthetic: alias.Global, unknown: alias.Unknown, fresh: alias.Fresh, binding: -1})
	}
	r := reference{place: Place{Root: id, Path: alias.Path}, loan: -1, result: index}
	if alias.Shared || alias.Global || alias.Unknown || seen[alias.FreshGroup] {
		r = e.borrow(r, point)
	}
	seen[alias.FreshGroup] = true
	return r
}

// builtin models standard operations that read or write native storage.
func (e *engine) builtin(point int, s state, call *ssa.CallCommon, name string) []reference {
	if name == "new" {
		return nil
	} // The SSA builder lowers new to Alloc.
	if name == "recover" {
		return e.unknownResults(point, call)
	}
	var result []reference
	for index, arg := range call.Args {
		for _, r := range e.refs(s, arg) {
			write := index == 0 && (name == "copy" || name == "clear" || name == "append")
			e.use(point, call.Pos(), r, write)
			if name == "panic" {
				e.unsupportedRetention(r, call.Pos(), "unsupported borrowed reference escaped through panic")
			}
			if name == "append" && index == 0 {
				result = append(result, e.borrow(r, point))
			}
		}
	}
	return result
}

// escape adds a caller-visible capture or a delayed-use requirement to a loan.
func (e *engine) escape(point int, r reference, deferred bool) {
	if !e.record {
		return
	}
	e.use(point, e.instructions[point].Pos(), r, false)
	root := e.roots[r.place.Root]
	if root.input >= 0 && !deferred {
		e.summary.Inputs[root.input].Capture = true
	}
	if !deferred {
		e.captured[r.place.Root] = true
	}
	if r.loan >= 0 {
		if deferred {
			e.loans[r.loan].deferred = true
		} else {
			e.loans[r.loan].external = true
		}
	}
}

// returned records origin relationships instead of treating every returned pointer as fresh.
func (e *engine) returned(point int, s state, op *ssa.Return) {
	if !e.record {
		return
	}
	for i, value := range op.Results {
		for _, r := range e.refs(s, value) {
			result := e.returnResult(point, s, op.Pos(), r)
			if !containsResult(e.summary.Results[i], result) {
				e.summary.Results[i] = append(e.summary.Results[i], result)
			}
		}
	}
}

// returnResult classifies one output as a parameter alias, shared global, or local allocation.
func (e *engine) returnResult(point int, s state, pos token.Pos, r reference) Result {
	e.use(point, pos, r, false)
	origin := e.roots[r.place.Root]
	if r.closure != nil {
		e.returnedClosure(point, s, r)
	}
	result := Result{Input: origin.input, Path: r.place.Path, Protected: origin.protected}
	if origin.unknown {
		result.Unknown = true
		result.Shared = true
	}
	if origin.input < 0 && !origin.global && !origin.unknown {
		result.Fresh = true
		result.FreshID = r.place.Root
	}
	if origin.global {
		result.Shared = true
		result.Global = true
		result.FreshID = r.place.Root
	}
	return result
}

// sameResult compares portable origin summaries without depending on slice identity.
func sameResult(a, b Result) bool {
	if a.Input != b.Input || a.FreshGroup != b.FreshGroup || a.Fresh != b.Fresh || a.FreshID != b.FreshID ||
		a.Shared != b.Shared || a.Protected != b.Protected || a.Global != b.Global || a.Unknown != b.Unknown {
		return false
	}
	return Place{Path: a.Path}.key() == (Place{Path: b.Path}).key()
}

// closure preserves each captured reference and turns address captures into shared loans.
func (e *engine) closure(point int, s state, op *ssa.MakeClosure) {
	fn := op.Fn.(*ssa.Function)
	var refs []reference
	for i, binding := range op.Bindings {
		for _, r := range e.refs(s, binding) {
			r = e.borrow(r, point)
			e.use(point, op.Pos(), r, false)
			r.closure = fn
			r.binding = i
			refs = append(refs, r)
		}
	}
	if len(refs) == 0 {
		id := e.root(op, types.NewSignatureType(nil, nil, nil, nil, nil, false), false, -1, false)
		refs = []reference{{place: Place{Root: id}, loan: -1, closure: fn, binding: -1}}
	}
	s.values[op] = refs
}

// resolvePath substitutes symbolic pointer dereferences with the caller's current memory provenance.
func (e *engine) resolvePath(s state, base reference, path []Projection) []reference {
	refs := []reference{base}
	for _, projection := range path {
		var next []reference
		for _, r := range refs {
			if projection.Kind != 'd' {
				r.place = r.place.project(projection.Kind, projection.Index)
				next = append(next, r)
				continue
			}
			found := false
			for _, stored := range s.cells {
				if Overlap(r.place, stored.place, false) {
					next, _ = mergeReferences(next, stored.refs)
					found = true
				}
			}
			if !found {
				r.place = r.place.project('d', 0)
				next = append(next, r)
			}
		}
		refs = next
	}
	return refs
}

// reachableUses keeps indirect loads of captured pointer variables live through helper calls.
func (e *engine) reachableUses(point int, pos token.Pos, s state, start reference, write bool) {
	pending := []reference{start}
	seen := make(map[string]bool)
	for len(pending) > 0 {
		r := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		key := r.place.key()
		if seen[key] {
			continue
		}
		seen[key] = true
		for _, stored := range s.cells {
			if Overlap(r.place, stored.place, false) {
				for _, target := range stored.refs {
					e.use(point, pos, target, write)
					pending = append(pending, target)
				}
			}
		}
	}
}

// unknownResults keeps unavailable reference results shared instead of erasing their provenance.
func (e *engine) unknownResults(point int, call *ssa.CallCommon) []reference {
	var refs []reference
	for index := 0; index < call.Signature().Results().Len(); index++ {
		typ := call.Signature().Results().At(index).Type()
		if !referenceType(typ) {
			continue
		}
		key := fmt.Sprintf("unknown:%d:%d", point, index)
		id, ok := e.callRoots[key]
		if !ok {
			id = len(e.roots)
			e.callRoots[key] = id
			protected := potentialInterior(storageType(typ))
			e.roots = append(e.roots, root{typ: storageType(typ), protected: protected, unknown: true, input: -1, binding: -1})
		}
		if e.roots[id].protected {
			e.report(call.Pos(), "unsupported call effects for an unknown reference result")
		}
		refs = append(refs, reference{place: Place{Root: id}, loan: -1, shared: true, result: index})
	}
	return refs
}

// potentialInterior identifies result shapes that could alias marked value storage.
func potentialInterior(t types.Type) bool { return potentialInteriorSeen(t, make(map[types.Type]bool)) }

// potentialInteriorSeen terminates recursive pointer shapes while looking for protected scalar leaves.
func potentialInteriorSeen(t types.Type, seen map[types.Type]bool) bool {
	if seen[t] {
		return false
	}
	seen[t] = true
	if checker.Contains(t) {
		return true
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		return true
	case *types.Array:
		return potentialInteriorSeen(u.Elem(), seen)
	case *types.Slice:
		return potentialInteriorSeen(u.Elem(), seen)
	case *types.Map:
		return potentialInteriorSeen(u.Key(), seen) || potentialInteriorSeen(u.Elem(), seen)
	case *types.Chan:
		return potentialInteriorSeen(u.Elem(), seen)
	case *types.Pointer:
		return potentialInteriorSeen(u.Elem(), seen)
	}
	return false
}

// returnedClosure exposes unmodelled closure environments in summaries instead of losing captured loans.
func (e *engine) returnedClosure(point int, s state, start reference) {
	pending := []reference{start}
	seen := make(map[string]bool)
	for len(pending) > 0 {
		r := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		key := r.place.key()
		if seen[key] {
			continue
		}
		seen[key] = true
		e.unsupportedRetention(r, e.instructions[point].Pos(), "unsupported returned closure carrying borrowed storage")
		for _, stored := range s.cells {
			if Overlap(r.place, stored.place, false) {
				pending = append(pending, stored.refs...)
			}
		}
	}
}

// containsResult checks an exact output alternative before exporting a portable summary.
func containsResult(results []Result, wanted Result) bool {
	for _, result := range results {
		if sameResult(result, wanted) {
			return true
		}
	}
	return false
}

// libraryResults models independent native copies and the fixed read-only iterator implementation.
func (e *engine) libraryResults(point int, call *ssa.CallCommon, callee *ssa.Function) []reference {
	var result []reference
	for index := 0; index < call.Signature().Results().Len(); index++ {
		typ := call.Signature().Results().At(index).Type()
		switch typ.Underlying().(type) {
		case *types.Slice, *types.Map, *types.Signature:
		default:
			continue
		}
		key := fmt.Sprintf("library:%d:%d", point, index)
		id, ok := e.callRoots[key]
		if !ok {
			id = len(e.roots)
			e.callRoots[key] = id
			e.roots = append(e.roots, root{typ: typ, protected: checker.Contains(typ), input: -1, fresh: true, binding: -1})
		}
		r := reference{place: Place{Root: id}, loan: -1, result: index}
		if _, iterator := typ.Underlying().(*types.Signature); iterator {
			r.closure = callee
			r.binding = -1
		}
		result = append(result, r)
	}
	return result
}
