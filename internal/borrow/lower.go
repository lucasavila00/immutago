package borrow

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lucasavila00/immutago/internal/borrow/flow"
	"golang.org/x/tools/go/cfg"
)

// build expands statement blocks into ordered use, write, and binding-definition points.
func (l *local) build(graph *cfg.CFG) {
	starts := make(map[*cfg.Block]int)
	ends := make(map[*cfg.Block]int)
	for _, block := range graph.Blocks {
		if !block.Live {
			continue
		}
		starts[block] = len(l.operations)
		l.operations = append(l.operations, operation{})
		l.rangeAssignment(block)
		for _, node := range block.Nodes {
			l.lower(node)
		}
		ends[block] = len(l.operations) - 1
	}
	successors := make([][]int, len(l.operations))
	for _, block := range graph.Blocks {
		if !block.Live {
			continue
		}
		for point := starts[block]; point < ends[block]; point++ {
			successors[point] = []int{point + 1}
		}
		for _, next := range block.Succs {
			if next.Live {
				successors[ends[block]] = append(successors[ends[block]], starts[next])
			}
		}
	}
	l.graph = flow.NewGraph(successors)
}

// lower preserves Go's evaluate-then-assign ordering without treating assignment targets as reads.
func (l *local) lower(node ast.Node) {
	switch statement := node.(type) {
	case *ast.AssignStmt:
		l.assign(statement.Lhs, statement.Rhs)
	case *ast.ValueSpec:
		targets := make([]ast.Expr, len(statement.Names))
		for i, name := range statement.Names {
			targets[i] = name
		}
		l.assign(targets, statement.Values)
	case *ast.DeclStmt:
		if declaration, ok := statement.Decl.(*ast.GenDecl); ok {
			for _, spec := range declaration.Specs {
				if value, ok := spec.(*ast.ValueSpec); ok {
					l.lower(value)
				}
			}
		}
	case *ast.IncDecStmt:
		l.read(statement.X)
		l.operations = append(l.operations, operation{writes: []ast.Expr{statement.X}})
	default:
		l.read(node)
	}
}

// assign records reference declarations and evaluates values before owner writes.
func (l *local) assign(targets, values []ast.Expr) {
	for _, target := range targets {
		l.readAddress(target)
	}
	for _, value := range values {
		l.read(value)
	}
	op := operation{}
	for index, target := range targets {
		variable := l.variable(target)
		if variable != nil && protectedPointer(variable.Type()) {
			var value ast.Expr
			if index < len(values) {
				value = values[index]
			}
			op.definitions = append(op.definitions, variable)
			l.reference(target, value, variable)
		} else {
			op.writes = append(op.writes, target)
		}
	}
	l.operations = append(l.operations, op)
}

// readAddress evaluates pointer and index operands on an assignment's left side.
func (l *local) readAddress(expression ast.Expr) {
	switch expression := ast.Unparen(expression).(type) {
	case *ast.SelectorExpr:
		l.read(expression.X)
	case *ast.IndexExpr:
		l.read(expression.X)
		l.read(expression.Index)
	case *ast.StarExpr:
		l.read(expression.X)
	}
}

// read records actual pointer-value uses while excluding nested function bodies.
func (l *local) read(node ast.Node) {
	uses := make(map[*types.Var]token.Pos)
	ast.Inspect(node, func(node ast.Node) bool {
		if _, ok := node.(*ast.FuncLit); ok {
			return false
		}
		ident, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		variable, ok := l.pass.TypesInfo.Uses[ident].(*types.Var)
		if ok && protectedPointer(variable.Type()) {
			uses[variable] = ident.Pos()
		}
		return true
	})
	l.operations = append(l.operations, operation{uses: uses})
}

// rangeAssignment models updates to existing range variables at every iteration.
func (l *local) rangeAssignment(block *cfg.Block) {
	loop, ok := block.Stmt.(*ast.RangeStmt)
	if !ok || block.Kind != cfg.KindRangeBody || loop.Tok != token.ASSIGN {
		return
	}
	var targets []ast.Expr
	if loop.Key != nil {
		targets = append(targets, loop.Key)
	}
	if loop.Value != nil {
		targets = append(targets, loop.Value)
	}
	l.assign(targets, nil)
}

// reference records the fixed owner of one local address declaration.
func (l *local) reference(target, source ast.Expr, pointer *types.Var) {
	id, ok := ast.Unparen(target).(*ast.Ident)
	if !ok || l.pass.TypesInfo.Defs[id] != pointer || source == nil {
		return
	}
	address, ok := ast.Unparen(source).(*ast.UnaryExpr)
	if !ok || address.Op != token.AND {
		return
	}
	owner, shared := l.owner(address.X)
	if owner != nil && !shared {
		l.pointers[pointer] = owner
	}
}
