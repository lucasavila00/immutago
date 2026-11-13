package freeze

import (
	"cmp"
	"go/ast"
	"go/token"
	"go/types"
	"maps"
	"slices"

	"github.com/lucasavila00/immutago/internal/borrow/flow"
	"golang.org/x/tools/go/cfg"
)

// event orders native owner uses, consuming calls, and fresh declaration resets.
type event struct {
	variable       *types.Var
	pos            token.Pos
	consume, reset bool
}

// checkFlow builds only functions participating in native ownership.
func (p *proof) checkFlow() {
	bodies := make(map[*ast.BlockStmt]bool)
	for _, item := range p.owners {
		if item.body != nil {
			bodies[item.body] = true
		}
	}
	ordered := make([]*ast.BlockStmt, 0, len(bodies))
	for body := range bodies {
		ordered = append(ordered, body)
	}
	slices.SortFunc(ordered, func(a, b *ast.BlockStmt) int { return cmp.Compare(a.Pos(), b.Pos()) })
	for _, body := range ordered {
		p.functionFlow(body)
	}
}

// functionFlow applies a may-consumed forward analysis, so a consumed incoming branch rejects later uses.
func (p *proof) functionFlow(body *ast.BlockStmt) {
	graph := cfg.New(body, func(*ast.CallExpr) bool { return true })
	var events []event
	starts, ends := make(map[*cfg.Block]int), make(map[*cfg.Block]int)
	for _, block := range graph.Blocks {
		if !block.Live {
			continue
		}
		starts[block] = len(events)
		events = append(events, event{})
		for _, node := range block.Nodes {
			events = append(events, p.events(node)...)
		}
		ends[block] = len(events) - 1
	}
	successors := make([][]int, len(events))
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
	p.consumption(flow.NewGraph(successors), events)
}

// consumption treats fresh declarations as new allocations even when the declaration executes in a loop.
func (p *proof) consumption(graph flow.Graph, events []event) {
	ids := make(map[*types.Var]int)
	for variable := range p.owners {
		ids[variable] = len(ids)
	}
	gen, kill := make([]flow.Set, len(events)), make([]flow.Set, len(events))
	for point, e := range events {
		gen[point], kill[point] = flow.NewSet(), flow.NewSet()
		if e.consume {
			gen[point].Add(ids[e.variable])
		}
		if e.reset {
			kill[point].Add(ids[e.variable])
		}
	}
	incoming := consumedFlow(graph, gen, kill)
	for point, e := range events {
		if e.variable != nil && !e.consume && !e.reset && incoming[point][ids[e.variable]] {
			p.report(e.pos, "use of consumed owned container "+e.variable.Name()+
				": Freeze transferred its storage; use the frozen value, or call Clone to create a new builder")
		}
	}
}

// events traverses expressions in Go evaluation order and consumes only after a Freeze argument is evaluated.
func (p *proof) events(node ast.Node) []event {
	var result []event
	var stack []ast.Node
	ast.Inspect(node, func(node ast.Node) bool {
		if node == nil {
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			result = append(result, p.finished(last)...)
			return false
		}
		if _, nested := node.(*ast.FuncLit); nested {
			return false
		}
		stack = append(stack, node)
		if id, ok := node.(*ast.Ident); ok {
			variable, _ := p.pass.TypesInfo.Uses[id].(*types.Var)
			if _, owned := p.owners[variable]; owned {
				result = append(result, event{variable: variable, pos: id.Pos()})
			}
		}
		return true
	})
	return result
}

// finished emits post-evaluation effects, preserving source argument and assignment ordering.
func (p *proof) finished(node ast.Node) []event {
	var result []event
	switch node := node.(type) {
	case *ast.IndexExpr:
		result = append(result, p.storageAccess(node.X)...)
	case *ast.CallExpr:
		result = append(result, p.builtinWrite(node)...)
		result = append(result, p.consumeCall(node)...)
	case *ast.ValueSpec:
		for _, id := range node.Names {
			result = append(result, p.reset(id)...)
		}
	case *ast.IncDecStmt:
		result = append(result, p.storageAccess(node.X)...)
	case *ast.AssignStmt:
		for _, target := range node.Lhs {
			result = append(result, p.storageAccess(target)...)
		}
		if node.Tok == token.DEFINE {
			for _, left := range node.Lhs {
				if id, ok := left.(*ast.Ident); ok {
					result = append(result, p.reset(id)...)
				}
			}
		}
	}
	return result
}

// reset recognizes only compiler definitions whose initializers established fresh ownership.
func (p *proof) reset(id *ast.Ident) []event {
	variable, _ := p.pass.TypesInfo.Defs[id].(*types.Var)
	item, owned := p.owners[variable]
	if owned && item.fresh {
		return []event{{variable: variable, pos: id.Pos(), reset: true}}
	}
	return nil
}

// consumedFlow solves the finite forward may-consumed equations in reverse postorder.
func consumedFlow(graph flow.Graph, gen, kill []flow.Set) []flow.Set {
	inputs, outputs := make([]flow.Set, len(gen)), make([]flow.Set, len(gen))
	order := graph.ReversePostorder()
	for changed := true; changed; {
		changed = false
		for _, point := range order {
			input := flow.NewSet()
			for _, previous := range graph.Pred[point] {
				for id := range outputs[previous] {
					input[id] = true
				}
			}
			output := maps.Clone(input)
			for id := range kill[point] {
				delete(output, id)
			}
			for id := range gen[point] {
				output[id] = true
			}
			if !maps.Equal(output, outputs[point]) {
				changed = true
			}
			inputs[point], outputs[point] = input, output
		}
	}
	return inputs
}

// storageAccess records an element access or delayed store after its operand evaluation completes.
func (p *proof) storageAccess(expression ast.Expr) []event {
	switch expression := ast.Unparen(expression).(type) {
	case *ast.Ident:
		variable, _ := p.pass.TypesInfo.Uses[expression].(*types.Var)
		if _, owned := p.owners[variable]; owned {
			return []event{{variable: variable, pos: expression.Pos()}}
		}
	case *ast.IndexExpr:
		return p.storageAccess(expression.X)
	case *ast.SelectorExpr:
		return p.storageAccess(expression.X)
	}
	return nil
}

// builtinWrite places native mutation after argument evaluation, not at the first handle read.
func (p *proof) builtinWrite(call *ast.CallExpr) []event {
	builtin, ok := calledObject(p.pass, call.Fun).(*types.Builtin)
	if !ok || len(call.Args) == 0 {
		return nil
	}
	switch builtin.Name() {
	case "append", "copy", "clear", "delete":
		return p.storageAccess(call.Args[0])
	}
	return nil
}

// consumeCall creates a post-call ownership transfer for a direct native owner argument.
func (p *proof) consumeCall(call *ast.CallExpr) []event {
	if !freezing(p.pass, call) || len(call.Args) != 1 {
		return nil
	}
	id, ok := ast.Unparen(call.Args[0]).(*ast.Ident)
	if !ok {
		return nil
	}
	variable, _ := p.pass.TypesInfo.Uses[id].(*types.Var)
	if _, owned := p.owners[variable]; owned {
		return []event{{variable: variable, pos: call.Pos(), consume: true}}
	}
	return nil
}
