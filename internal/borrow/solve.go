// Copyright (c) The Rust Project Contributors
// Local liveness uses rustc's backward use-to-definition traversal through flow.BackwardLive.
// Source revision and license: third_party/rustc.

package borrow

import (
	"go/ast"
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
				if definition == pointer {
					definitions.Add(point)
				}
			}
		}
		live[pointer] = flow.BackwardLive(l.graph, uses, definitions)
	}
	return live
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
func (l *local) checkWrite(point int, expression ast.Expr, live map[*types.Var]flow.Set) {
	owner, shared := l.owner(expression)
	if shared {
		l.sharedWrite(expression, owner)
		return
	}
	if owner == nil || !checker.Contains(owner.Type()) {
		return
	}
	if owner.Pkg() != nil && owner.Parent() == owner.Pkg().Scope() {
		l.globalWrite(expression, owner)
		return
	}
	var conflict *types.Var
	for pointer, borrowed := range l.pointers {
		if live[pointer][point] && borrowed == owner && (conflict == nil || pointer.Pos() < conflict.Pos()) {
			conflict = pointer
		}
	}
	if conflict != nil {
		l.borrowedWrite(point, expression, owner, conflict)
	}
}
