// Copyright (c) The Rust Project Contributors
// Local liveness uses rustc's backward use-to-definition traversal through flow.BackwardLive.
// Source revision and license: third_party/rustc.
package borrow

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lucasavila00/immutago/internal/borrow/flow"
	"github.com/lucasavila00/immutago/internal/checker"
)

// liveness computes the points where each current pointer binding can still be read.
func (l *local) liveness() map[*types.Var]flow.Set {
	live := make(map[*types.Var]flow.Set)
	for pointer := range l.pointers {
		uses, definitions := flow.NewSet(), flow.NewSet()
		for point, op := range l.operations {
			if op.uses[pointer].IsValid() {
				uses.Add(point)
			}
			for _, definition := range op.definitions {
				if definition.target == pointer {
					definitions.Add(point)
				}
			}
		}
		live[pointer] = flow.BackwardLive(l.graph, uses, definitions)
	}
	return live
}

// reaching computes finite may-origins, strongly replacing each reassigned pointer binding.
func (l *local) reaching() []bindings {
	incoming := make([]bindings, len(l.operations))
	outgoing := make([]bindings, len(l.operations))
	order := l.graph.ReversePostorder()
	for changed := true; changed; {
		changed = false
		for _, point := range order {
			in := make(bindings)
			if point == 0 {
				mergeBindings(in, l.parameters)
			}
			for _, previous := range l.graph.Pred[point] {
				mergeBindings(in, outgoing[previous])
			}
			out := make(bindings)
			mergeBindings(out, in)
			for _, definition := range l.operations[point].definitions {
				out[definition.target] = l.origins(definition.source, in)
			}
			if !sameBindings(outgoing[point], out) {
				changed = true
			}
			incoming[point], outgoing[point] = in, out
		}
	}
	return incoming
}

// mergeBindings joins each possible whole owner without sharing mutable sets between points.
func mergeBindings(target, source bindings) {
	for pointer, owners := range source {
		if target[pointer] == nil {
			target[pointer] = make(map[*types.Var]bool)
		}
		for owner := range owners {
			target[pointer][owner] = true
		}
	}
}

// sameBindings compares finite owner sets independent of map iteration order.
func sameBindings(left, right bindings) bool {
	if len(left) != len(right) {
		return false
	}
	for pointer, owners := range left {
		if len(owners) != len(right[pointer]) {
			return false
		}
		for owner := range owners {
			if !right[pointer][owner] {
				return false
			}
		}
	}
	return true
}

// origins resolves local pointer copies and address expressions without interprocedural guesses.
func (l *local) origins(expression ast.Expr, incoming bindings) map[*types.Var]bool {
	if expression == nil {
		return nil
	}
	expression = ast.Unparen(expression)
	if pointer := l.variable(expression); pointer != nil && protectedPointer(pointer.Type()) {
		return incoming[pointer]
	}
	if address, ok := expression.(*ast.UnaryExpr); ok && address.Op == token.AND {
		if variable, shared := l.owner(address.X); variable != nil {
			if shared {
				return incoming[variable]
			}
			return map[*types.Var]bool{variable: true}
		}
		return nil
	}

	return nil
}

// owner reduces a field or array projection to its whole variable and shared-pointer access flag.
func (l *local) owner(expression ast.Expr) (*types.Var, bool) {
	expression = ast.Unparen(expression)
	if variable := l.variable(expression); variable != nil {
		return variable, protectedPointer(variable.Type())
	}
	switch expression := expression.(type) {
	case *ast.SelectorExpr:
		if l.pass.TypesInfo.Selections[expression] == nil {
			variable, _ := l.pass.TypesInfo.Uses[expression.Sel].(*types.Var)
			return variable, false
		}
		variable, shared := l.owner(expression.X)
		return variable, shared || protectedPointer(l.pass.TypesInfo.TypeOf(expression.X))
	case *ast.IndexExpr:
		return l.owner(expression.X)
	case *ast.StarExpr:
		variable, _ := l.owner(expression.X)
		return variable, protectedPointer(l.pass.TypesInfo.TypeOf(expression.X))
	}
	return nil, false
}

// checkWrite rejects shared writes and writes to a whole owner with any live local alias.
func (l *local) checkWrite(point int, expression ast.Expr, incoming bindings, live map[*types.Var]flow.Set) {
	owner, shared := l.owner(expression)
	if shared {
		l.sharedWrite(expression, owner)
		return
	}
	if owner == nil || !checker.Contains(owner.Type()) {
		return
	}
	if owner.Pkg() != nil && owner.Parent() == owner.Pkg().Scope() && !l.initializer {
		l.globalWrite(expression, owner)
		return
	}
	wanted := owner
	for _, definition := range l.operations[point].definitions {
		if l.origins(definition.source, incoming)[wanted] {
			l.borrowedWrite(point, expression, owner, definition.target, definition.source.Pos())
			return
		}
	}
	var conflict *types.Var
	for pointer, owners := range incoming {
		if live[pointer][point] && owners[wanted] && (conflict == nil || pointer.Pos() < conflict.Pos()) {
			conflict = pointer
		}
	}
	if conflict != nil {
		l.borrowedWrite(point, expression, owner, conflict, token.NoPos)
	}
}
