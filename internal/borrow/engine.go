package borrow

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"

	"github.com/lucasavila00/immutago/internal/borrow/flow"
	"github.com/lucasavila00/immutago/internal/checker"
	"golang.org/x/tools/go/ssa"
)

// newEngine gives every compiler instruction an ordered point and retains its CFG edges.
func newEngine(p *program, fn *ssa.Function, diagnose bool) *engine {
	e := &engine{program: p, fn: fn, diagnose: diagnose, allocations: make(map[ssa.Value]int),
		points: make(map[ssa.Instruction]int), loanIDs: make(map[string]int), captured: make(map[int]bool),
		callRoots: make(map[string]int)}
	for _, block := range fn.Blocks {
		for _, instruction := range block.Instrs {
			e.points[instruction] = len(e.instructions)
			e.instructions = append(e.instructions, instruction)
		}
	}
	successors := make([][]int, len(e.instructions))
	for _, block := range fn.Blocks {
		for i, instruction := range block.Instrs {
			point := e.points[instruction]
			if i+1 < len(block.Instrs) {
				successors[point] = []int{point + 1}
				continue
			}
			for _, next := range block.Succs {
				if len(next.Instrs) > 0 {
					successors[point] = append(successors[point], e.points[next.Instrs[0]])
				}
			}
		}
	}
	e.graph = flow.NewGraph(successors)
	e.inputs = make([]state, len(successors))
	e.outputs = make([]state, len(successors))
	for i := range successors {
		e.inputs[i] = emptyState()
		e.outputs[i] = emptyState()
	}
	e.summary.Inputs = make([]Effect, len(fn.Params)+len(fn.FreeVars))
	e.summary.Results = make([][]Result, fn.Signature.Results().Len())
	return e
}

// run resolves aliases to a fixed point, records effects, then checks loans at accesses.
func (e *engine) run() Summary {
	if len(e.instructions) == 0 {
		return e.summary
	}
	initial := e.initial()
	order := e.graph.ReversePostorder()
	for changed := true; changed; {
		changed = false
		for _, point := range order {
			in := emptyState()
			if point == 0 {
				in.join(initial)
			}
			for _, pred := range e.graph.Pred[point] {
				in.join(e.outputs[pred])
			}
			out := in.clone()
			e.step(point, out)
			if !sameState(e.outputs[point], out) {
				changed = true
			}
			e.inputs[point] = in
			e.outputs[point] = out
		}
	}
	e.record = true
	for _, point := range order {
		e.step(point, e.inputs[point].clone())
	}
	if e.diagnose {
		e.checkLoans()
	}
	e.normalizeFresh()

	return e.summary
}

// initial assigns symbolic roots to pointer and slice parameters and captured variables.
func (e *engine) initial() state {
	result := emptyState()
	for i, param := range e.fn.Params {
		e.parameter(result, param, i)
	}
	for i, param := range e.fn.FreeVars {
		e.parameter(result, param, len(e.fn.Params)+i)
	}
	return result
}

// parameter preserves a function input as a shared origin for effect inference.
func (e *engine) parameter(s state, value ssa.Value, index int) {
	if !referenceType(value.Type()) {
		return
	}
	id := e.root(value, storageType(value.Type()), false, index, false)
	s.values[value] = []reference{{place: Place{Root: id}, loan: -1, shared: true}}
}

// root assigns a stable identity to storage represented by an SSA value.
func (e *engine) root(value ssa.Value, t types.Type, global bool, input int, fresh bool) int {
	if id, ok := e.allocations[value]; ok {
		return id
	}
	id := len(e.roots)
	e.allocations[value] = id
	e.roots = append(e.roots, root{typ: t, protected: checker.Contains(t), global: global,
		input: input, fresh: fresh, binding: -1})
	return id
}

// refs reads a compiler value's provenance, creating global slot identities on demand.
func (e *engine) refs(s state, value ssa.Value) []reference {
	if fn, ok := value.(*ssa.Function); ok {
		id := e.root(fn, fn.Type(), false, -1, false)
		e.roots[id].protected = callableCapability(fn.Type())
		return []reference{{place: Place{Root: id}, loan: -1, closure: fn, binding: -1}}
	}
	if global, ok := value.(*ssa.Global); ok {
		id := e.root(global, storageType(global.Type()), true, -1, false)
		return []reference{{place: Place{Root: id}, loan: -1}}
	}
	return s.values[value]
}

// storageType removes one pointer layer while preserving a slice's backing-storage type.
func storageType(t types.Type) types.Type {
	if ptr, ok := t.Underlying().(*types.Pointer); ok {
		return ptr.Elem()
	}
	return t
}

// referenceType identifies values that can retain an alias, including erased pointers and closures.
func referenceType(t types.Type) bool {
	switch u := t.Underlying().(type) {
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan, *types.Interface, *types.Signature:
		return true
	case *types.Array:
		return referenceType(u.Elem())
	}
	return false
}

// sameState compares finite may-alias sets without depending on map iteration order.
func sameState(a, b state) bool {
	if len(a.values) != len(b.values) || len(a.cells) != len(b.cells) {
		return false
	}
	for k, v := range a.values {
		if !sameReferences(v, b.values[k]) {
			return false
		}
	}
	for k, v := range a.cells {
		if !sameReferences(v.refs, b.cells[k].refs) {
			return false
		}
	}
	return true
}

// sameReferences compares two unordered provenance sets.
func sameReferences(a, b []reference) bool {
	if len(a) != len(b) {
		return false
	}
	for _, r := range a {
		found := false
		for _, s := range b {
			if sameReference(r, s) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// borrow creates a shared loan while preserving an existing shared loan through copies.
func (e *engine) borrow(r reference, point int) reference {
	if r.shared {
		return r
	}
	key := fmt.Sprintf("%d:%s", point, r.place.key())
	index, ok := e.loanIDs[key]
	if !ok {
		index = len(e.loans)
		e.loanIDs[key] = index
		e.loans = append(e.loans, loan{place: r.place, point: point, uses: flow.NewSet()})
	}
	r.loan = index
	r.shared = true
	return r
}

// use records an access and its loan-liveness seed after provenance has converged.
func (e *engine) use(point int, pos token.Pos, r reference, write bool) {
	if !e.record || r.place.Root < 0 {
		return
	}
	if r.loan >= 0 {
		e.loans[r.loan].uses.Add(point)
	}
	origin := e.roots[r.place.Root]
	if origin.unknown && write {
		e.summary.Unproved = true
	}
	if origin.input >= 0 {
		effect := &e.summary.Inputs[origin.input]
		effect.Read = true
		effect.Write = effect.Write || write
	}
	e.accesses = append(e.accesses, access{point: point, pos: pos, ref: r, write: write})
}

// report emits a source diagnostic only during the final checked pass.
func (e *engine) report(pos token.Pos, message string) {
	if e.diagnose && e.record && pos.IsValid() {
		e.program.report(pos, message)
	}
}

// normalizeFresh bounds exported allocation identities by their result-position alias groups.
func (e *engine) normalizeFresh() {
	groups := make(map[int]string)
	for index, results := range e.summary.Results {
		for _, result := range results {
			if result.Fresh {
				part := fmt.Sprintf("%d,", index)
				if !strings.Contains(groups[result.FreshID], part) {
					groups[result.FreshID] += part
				}
			}
		}
	}
	for index, results := range e.summary.Results {
		for j, result := range results {
			if !result.Fresh {
				continue
			}
			result.FreshGroup = groups[result.FreshID]
			result.Shared = result.Shared || e.captured[result.FreshID]
			result.FreshID = 0
			e.summary.Results[index][j] = result
		}
	}
}

// callableCapability identifies callbacks whose reference results can expose protected storage.
// Interfaces and nested callbacks count because their concrete result can erase a protected alias.
func callableCapability(t types.Type) bool {
	signature, ok := t.Underlying().(*types.Signature)
	if !ok {
		return false
	}
	for i := 0; i < signature.Results().Len(); i++ {
		if referenceType(signature.Results().At(i).Type()) {
			return true
		}
	}
	return false
}
