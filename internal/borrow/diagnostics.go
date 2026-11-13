package borrow

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// report emits a stable rule message with compiler locations for supporting evidence.
func (l *local) report(pos token.Pos, message string, related ...analysis.RelatedInformation) {
	if l.reported[pos] {
		return
	}
	l.reported[pos] = true
	l.pass.Report(analysis.Diagnostic{Pos: pos, Message: message, Related: related})
}

// sharedWrite explains that a pointer never grants write authority to protected storage.
func (l *local) sharedWrite(expression ast.Expr, pointer *types.Var) {
	message := fmt.Sprintf("cannot write through shared reference %s: shared references are read-only; "+
		"update the owned value before borrowing, or copy the value and update the copy", types.ExprString(expression))
	var related []analysis.RelatedInformation
	if pointer != nil {
		related = append(related, analysis.RelatedInformation{Pos: pointer.Pos(),
			Message: fmt.Sprintf("shared reference %s (%s) declared here", pointer.Name(), typeName(pointer.Type()))})
	}
	l.report(expression.Pos(), message, related...)
}

// globalWrite points to the global declaration and its construction boundary.
func (l *local) globalWrite(expression ast.Expr, owner *types.Var) {
	message := fmt.Sprintf("cannot write package-level immutable storage %s (%s): initialize it before publication, "+
		"or copy it into a local value before updating", owner.Name(), typeName(owner.Type()))
	l.report(expression.Pos(), message, analysis.RelatedInformation{
		Pos: owner.Pos(), Message: "immutable global declared here"})
}

// borrowedWrite names the whole owner, live binding, and a reachable use keeping the loan alive.
func (l *local) borrowedWrite(point int, expression ast.Expr, owner, pointer *types.Var, borrow token.Pos) {
	message := fmt.Sprintf("cannot write borrowed immutable owner %s (%s) through %s: shared reference %s is still live; "+
		"move this write after its final use, or copy the value before updating",
		owner.Name(), typeName(owner.Type()), types.ExprString(expression), pointer.Name())
	related := []analysis.RelatedInformation{{Pos: pointer.Pos(),
		Message: "shared reference " + pointer.Name() + " declared here"}}
	if borrow.IsValid() {
		related = append(related, analysis.RelatedInformation{Pos: borrow,
			Message: "this right-hand side borrows the owner before the assignment writes it"})
	}
	if use := l.futureUse(point, pointer); use.IsValid() {
		related = append(related, analysis.RelatedInformation{Pos: use,
			Message: "this reachable use keeps shared reference " + pointer.Name() + " live"})
	}
	l.report(expression.Pos(), message, related...)
}

// futureUse finds supporting evidence along CFG edges before the binding is reassigned.
func (l *local) futureUse(point int, pointer *types.Var) token.Pos {
	pending := append([]int(nil), l.graph.Succ[point]...)
	seen := make(map[int]bool)
	for len(pending) > 0 {
		current := pending[0]
		pending = pending[1:]
		if seen[current] {
			continue
		}
		seen[current] = true
		op := l.operations[current]
		if use := op.uses[pointer]; use.IsValid() {
			return use
		}
		if defines(op, pointer) {
			continue
		}
		pending = append(pending, l.graph.Succ[current]...)
	}
	return token.NoPos
}

// defines checks whether an operation replaces the current pointer binding.
func defines(op operation, pointer *types.Var) bool {
	for _, definition := range op.definitions {
		if definition.target == pointer {
			return true
		}
	}
	return false
}

// typeName renders compiler types with familiar package names for diagnostics.
func typeName(t types.Type) string {
	return types.TypeString(t, func(pkg *types.Package) string { return pkg.Name() })
}
