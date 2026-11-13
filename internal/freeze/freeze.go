// Package freeze proves exclusive native container ownership before zero-copy freezing.
package freeze

import (
	"cmp"
	"go/ast"
	"go/token"
	"go/types"
	"slices"

	"github.com/lucasavila00/immutago/internal/checker"
	"golang.org/x/tools/go/analysis"
)

// owner records the sole creation site of a mutable native container.
type owner struct {
	variable *types.Var
	body     *ast.BlockStmt
	fresh    bool
}

// proof keeps ownership discovery separate from alias restrictions and consumed-state flow.
type proof struct {
	pass         *analysis.Pass
	parents      map[ast.Node]ast.Node
	declarations map[*types.Var]owner
	owners       map[*types.Var]owner
	reported     map[token.Pos]bool
}

// Check discovers owned builders, rejects aliases, and checks every use after consumption.
// The returned set lets the analyzer admit these native containers without exempting their operations.
func Check(pass *analysis.Pass) map[*types.Var]bool {
	p := &proof{pass: pass, parents: make(map[ast.Node]ast.Node), declarations: make(map[*types.Var]owner),
		owners: make(map[*types.Var]owner), reported: make(map[token.Pos]bool)}
	p.discover()
	result := make(map[*types.Var]bool)
	variables := make([]*types.Var, 0, len(p.owners))
	for variable := range p.owners {
		variables = append(variables, variable)
	}
	slices.SortFunc(variables, func(a, b *types.Var) int { return cmp.Compare(a.Pos(), b.Pos()) })
	for _, variable := range variables {
		item := p.owners[variable]
		result[variable] = true
		if item.body == nil || !item.fresh {
			p.report(variable.Pos(), "cannot freeze "+variable.Name()+
				": ownership requires a fresh local make, literal, nil, or Clone result; create a local builder first")
		}
	}
	p.checkUses()
	p.checkFlow()
	return result
}

// Fresh recognizes only compiler-resolved allocation and copying operations.
func Fresh(pass *analysis.Pass, expression ast.Expr) bool {
	if expression == nil {
		return false
	}
	expression = ast.Unparen(expression)
	if id, ok := expression.(*ast.Ident); ok && pass.TypesInfo.ObjectOf(id) == types.Universe.Lookup("nil") {
		return true
	}
	if !native(pass.TypesInfo.TypeOf(expression)) {
		return false
	}
	if _, ok := expression.(*ast.CompositeLit); ok {
		return true
	}
	call, ok := expression.(*ast.CallExpr)
	if !ok {
		return false
	}
	object := calledObject(pass, call.Fun)
	if builtin, ok := object.(*types.Builtin); ok {
		return builtin.Name() == "make"
	}
	if object == nil || object.Pkg() == nil || object.Pkg().Path() != checker.LibraryPath || object.Name() != "Clone" {
		return false
	}
	selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok || len(call.Args) != 0 {
		return false
	}
	selection := pass.TypesInfo.Selections[selector]
	return selection != nil && selection.Kind() == types.MethodVal && checker.IsCollection(selection.Recv())
}

// native identifies mutable native handles rather than value arrays or frozen wrappers.
func native(t types.Type) bool {
	if t == nil {
		return false
	}
	switch t.Underlying().(type) {
	case *types.Slice, *types.Map:
		return true
	}
	return false
}

// calledObject resolves generic instantiations and selector calls to their declared object.
func calledObject(pass *analysis.Pass, expression ast.Expr) types.Object {
	switch expression := ast.Unparen(expression).(type) {
	case *ast.Ident:
		return pass.TypesInfo.ObjectOf(expression)
	case *ast.SelectorExpr:
		return pass.TypesInfo.ObjectOf(expression.Sel)
	case *ast.IndexExpr:
		return calledObject(pass, expression.X)
	case *ast.IndexListExpr:
		return calledObject(pass, expression.X)
	}
	return nil
}

// freezing recognizes only the sealed library's two ownership-consuming entry points.
func freezing(pass *analysis.Pass, call *ast.CallExpr) bool {
	object := calledObject(pass, call.Fun)
	return object != nil && object.Pkg() != nil && object.Pkg().Path() == checker.LibraryPath &&
		(object.Name() == "FreezeSlice" || object.Name() == "FreezeMap")
}

// report emits one actionable ownership diagnostic at a source position.
func (p *proof) report(pos token.Pos, message string) {
	if !p.reported[pos] {
		p.reported[pos] = true
		p.pass.Reportf(pos, "%s", message)
	}
}
