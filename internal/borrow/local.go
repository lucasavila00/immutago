// Package borrow checks local shared loans with whole-owner nonlexical lifetimes.
// Its backward liveness solver is ported from rustc; see flow and third_party/rustc.
package borrow

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lucasavila00/immutago/internal/borrow/flow"
	"github.com/lucasavila00/immutago/internal/checker"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/cfg"
)

// assignment changes a pointer binding after all right-hand sides have been evaluated.
type assignment struct {
	target *types.Var
	source ast.Expr
}

// operation separates uses from definitions and writes at one source location.
type operation struct {
	uses        map[*types.Var]token.Pos
	definitions []assignment
	writes      []ast.Expr
}

// bindings maps local pointer variables to all owners reaching the current point.
type bindings map[*types.Var]map[*types.Var]bool

// local owns one function's finite source-level control-flow analysis.
type local struct {
	pass        *analysis.Pass
	operations  []operation
	graph       flow.Graph
	pointers    map[*types.Var]bool
	parameters  bindings
	initializer bool
	reported    map[token.Pos]bool
}

// Check checks function bodies locally without loading or inferring dependency bodies.
func Check(pass *analysis.Pass) error {
	if IsStandardLibrary(pass) {
		return nil
	}
	relevance := make(map[types.Type]bool)
	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			switch fn := node.(type) {
			case *ast.FuncDecl:
				if fn.Body != nil && protectedBody(pass, fn.Body, relevance) {
					checkFunction(pass, fn.Body, fn.Name.Name == "init")
				}
			case *ast.FuncLit:
				if protectedBody(pass, fn.Body, relevance) {
					checkFunction(pass, fn.Body, false)
				}
			}
			return true
		})
	}
	return nil
}

// checkFunction lowers one body and checks shared permissions and live whole-owner loans.
func checkFunction(pass *analysis.Pass, body *ast.BlockStmt, initializer bool) {
	l := &local{pass: pass, pointers: make(map[*types.Var]bool), parameters: make(bindings),
		initializer: initializer, reported: make(map[token.Pos]bool)}
	l.build(cfg.New(body, func(*ast.CallExpr) bool { return true }))
	if len(l.pointers) == 0 && !l.hasWrites() {
		return
	}
	incoming := l.reaching()
	live := l.liveness()
	for point, op := range l.operations {
		for _, expression := range op.writes {
			l.checkWrite(point, expression, incoming[point], live)
		}
	}
}

// hasWrites avoids solving empty functions while retaining direct pointer and global writes.
func (l *local) hasWrites() bool {
	for _, op := range l.operations {
		if len(op.writes) > 0 {
			return true
		}
	}
	return false
}

// UsesImmutable identifies marked-storage import dependencies without inferring their bodies.
func UsesImmutable(pkg *types.Package) bool {
	return importsImmutable(pkg, make(map[*types.Package]bool))
}

// importsImmutable visits each compiler package at most once.
func importsImmutable(pkg *types.Package, seen map[*types.Package]bool) bool {
	if pkg.Path() == checker.LibraryPath {
		return true
	}
	if seen[pkg] {
		return false
	}
	seen[pkg] = true
	for _, imported := range pkg.Imports() {
		if importsImmutable(imported, seen) {
			return true
		}
	}
	return false
}

// protectedPointer recognizes exactly one pointer to protected value storage.
func protectedPointer(t types.Type) bool {
	if t == nil {
		return false
	}
	pointer, ok := t.Underlying().(*types.Pointer)
	return ok && checker.Contains(pointer.Elem())
}

// variable resolves both declarations and uses to their compiler variable identity.
func (l *local) variable(expression ast.Expr) *types.Var {
	ident, ok := ast.Unparen(expression).(*ast.Ident)
	if !ok {
		return nil
	}
	object := l.pass.TypesInfo.ObjectOf(ident)
	variable, _ := object.(*types.Var)
	return variable
}

// protectedBody detects actual marked storage even when compiler export imports are pruned.
func protectedBody(pass *analysis.Pass, body *ast.BlockStmt, cache map[types.Type]bool) bool {
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		if found {
			return false
		}
		if _, nested := node.(*ast.FuncLit); nested {
			return false
		}
		expression, ok := node.(ast.Expr)
		if !ok {
			return true
		}
		typ := pass.TypesInfo.TypeOf(expression)
		if typ == nil {
			return true
		}
		protected, known := cache[typ]
		if !known {
			protected = checker.Contains(typ)
			cache[typ] = protected
		}
		found = protected
		return !found
	})
	return found
}
