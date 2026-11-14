package freeze

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
)

// discover indexes parents and all local definitions before selecting ownership participants.
func (p *proof) discover() {
	for _, file := range p.pass.Files {
		var stack []ast.Node
		ast.Inspect(file, func(node ast.Node) bool {
			if node == nil {
				stack = stack[:len(stack)-1]
				return false
			}
			if len(stack) > 0 {
				p.parents[node] = stack[len(stack)-1]
			}
			stack = append(stack, node)
			p.declaration(node)
			return true
		})
	}
	for variable, item := range p.declarations {
		if item.fresh && checker.Contains(variable.Type()) {
			p.owners[variable] = item
		}
	}
	for _, file := range p.pass.Files {
		ast.Inspect(file, p.freezeInput)
	}
}

// declare records initializers without treating later assignments as fresh ownership.
func (p *proof) declare(node ast.Node, names []*ast.Ident, values []ast.Expr, zero bool) {
	for index, name := range names {
		if name == nil {
			continue
		}
		variable, _ := p.pass.TypesInfo.Defs[name].(*types.Var)
		if variable == nil || !native(variable.Type()) {
			continue
		}
		item := owner{variable: variable, body: p.body(node), fresh: zero}
		if len(values) == len(names) {
			item.fresh = Fresh(p.pass, values[index])
		}
		p.declarations[variable] = item
	}
}

// body identifies the nearest function while parent indexing is already available for declarations.
func (p *proof) body(node ast.Node) *ast.BlockStmt {
	for node != nil {
		switch fn := node.(type) {
		case *ast.FuncDecl:
			return fn.Body
		case *ast.FuncLit:
			return fn.Body
		}
		node = p.parents[node]
	}
	return nil
}

// freezeInput requires a local owner or an immediately fresh allocation at every Freeze call.
func (p *proof) freezeInput(node ast.Node) bool {
	call, ok := node.(*ast.CallExpr)
	if !ok || !freezing(p.pass, call) || len(call.Args) != 1 {
		return true
	}
	if checker.Unevaluated(p.pass.TypesInfo, p.parents, p.unevaluated, call) {
		return true
	}
	expression := ast.Unparen(call.Args[0])
	if Fresh(p.pass, expression) {
		return true
	}
	if id, ok := expression.(*ast.Ident); ok {
		variable, _ := p.pass.TypesInfo.ObjectOf(id).(*types.Var)
		if variable != nil && native(variable.Type()) {
			item, found := p.declarations[variable]
			if !found {
				item = owner{variable: variable}
			}
			p.owners[variable] = item
			return true
		}
	}
	p.report(expression.Pos(), "cannot freeze this expression: pass an exclusively owned local builder "+
		"or a fresh make, literal, bare nil, or Clone result")
	return true
}

// declaration records one source declaration after its ancestors have been indexed.
func (p *proof) declaration(node ast.Node) {
	switch node := node.(type) {
	case *ast.ValueSpec:
		p.declare(node, node.Names, node.Values, len(node.Values) == 0)
	case *ast.AssignStmt:
		if node.Tok == token.DEFINE {
			var names []*ast.Ident
			for _, left := range node.Lhs {
				id, _ := left.(*ast.Ident)
				names = append(names, id)
			}
			p.declare(node, names, node.Rhs, false)
		}
	}
}
