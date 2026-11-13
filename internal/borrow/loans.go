// Algorithms adapted from rustc at 80196be0e787008b6fb14d2990aba92083cd68e1.
// Sources: rustc_borrowck/src/dataflow.rs and type_check/liveness/trace.rs.
// See third_party/rustc for attribution and license terms.

package borrow

import (
	"strings"

	"github.com/lucasavila00/immutago/internal/borrow/flow"
)

// checkLoans ports issuance, nonlexical region expiration, and forward loan checks.
// Region kills precede an access. Storage overwrite kills follow its conflict check.
func (e *engine) checkLoans() {
	gen, early, primary := e.loanSets(e.loanRegions())
	kill := make([]flow.Set, len(e.instructions))
	for i := range kill {
		kill[i] = early[i].Clone()
		kill[i].Union(primary[i])
	}
	incoming, _ := flow.Forward(e.graph, gen, kill)
	for _, a := range e.accesses {
		e.checkAccess(a, incoming[a.point].Without(early[a.point]))
	}
}

// loanRegions seeds actual reference uses and follows CFG predecessors to valid definition boundaries.
func (e *engine) loanRegions() []flow.Set {
	seeds := make([]flow.Set, len(e.loans))
	for index, l := range e.loans {
		uses := l.uses.Clone()
		if l.deferred || l.external {
			// Deferred reads can run on implicit panic edges; escaped readers can outlive this CFG.
			for _, point := range e.graph.ReversePostorder() {
				uses.Add(point)
			}
		}
		definitions := flow.NewSet(l.point)
		if e.carrierSurvives(index, l.point) {
			definitions = flow.NewSet()
		}
		seeds[index] = flow.BackwardLive(e.graph, uses, definitions)
		seeds[index].Add(l.point)
	}
	// Aliases carry their issuing loan ID, so their region constraints are already unified.
	return flow.SolveRegions(seeds, nil)
}

// loanSets separates early region expiration from primary overwrite kills.
func (e *engine) loanSets(regions []flow.Set) (gen, early, primary []flow.Set) {
	count := len(e.instructions)
	gen, early, primary = make([]flow.Set, count), make([]flow.Set, count), make([]flow.Set, count)
	for i := range count {
		gen[i] = flow.NewSet()
		early[i] = flow.NewSet()
		primary[i] = flow.NewSet()
	}
	for index, l := range e.loans {
		gen[l.point].Add(index)
		for point := range flow.OutOfScope(e.graph, l.point, regions[index]) {
			early[point].Add(index)
		}
	}
	for _, a := range e.accesses {
		if !a.write {
			continue
		}
		for index, l := range e.loans {
			if Overlap(a.ref.place, l.place, true) {
				primary[a.point].Add(index)
			}
		}
	}
	return gen, early, primary
}

// checkAccess rejects writes through shared references or across overlapping live loans.
func (e *engine) checkAccess(a access, active flow.Set) {
	if !a.write || !e.roots[a.ref.place.Root].protected {
		return
	}
	origin := e.roots[a.ref.place.Root]
	initializer := e.fn.Name() == "init" || strings.HasPrefix(e.fn.Name(), "init#")
	if origin.global && globallyProtected(origin.typ) && !initializer {
		e.report(a.pos, "cannot write package-level immutable storage")
		return
	}
	if a.ref.shared {
		e.report(a.pos, "cannot write through a shared reference")
		return
	}
	for index := range active {
		if Overlap(a.ref.place, e.loans[index].place, false) {
			e.report(a.pos, "cannot write borrowed storage while a shared reference is live")
			return
		}
	}
}

// carrierSurvives prevents a repeated issuance site from killing references from earlier loop iterations.
// A loan ends at a definition only when no other reference carrier remains live across that point.
func (e *engine) carrierSurvives(index, point int) bool {
	for _, stored := range e.inputs[point].cells {
		carries := false
		for _, r := range stored.refs {
			if r.loan == index {
				carries = true
				break
			}
		}
		if !carries {
			continue
		}
		uses, definitions := flow.NewSet(), flow.NewSet()
		for _, a := range e.accesses {
			if !Overlap(a.ref.place, stored.place, true) {
				continue
			}
			if a.write {
				definitions.Add(a.point)
			} else {
				uses.Add(a.point)
			}
		}
		if flow.BackwardLive(e.graph, uses, definitions)[point] {
			return true
		}
	}
	return false
}
