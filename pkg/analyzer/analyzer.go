// Package analyzer exposes the immutago go/analysis analyzer.
//
// Marked structs may own only primitive values, arrays of immutable values,
// other marked structs, and the library's Slice and Map types with immutable
// arguments. Unmarked structs cannot store marked values or these collections.
// Pointer fields, interfaces, functions, channels, and native mutable collections
// cannot establish an immutability proof. Reflection and unsafe operations are
// unsupported. Unmarked generic fields must prove their
// constraints exclude immutable storage. There are no suppression directives.
//
// Owned local values permit construction and field updates. Shared references
// permit reads; their loans end after the final required use on each control-flow
// path. Writes to overlapping borrowed storage are rejected. Value copies have
// independent storage. Package-level immutable values are read-only after init.
// Function summaries preserve aliases and effects across package boundaries.
// Analyze every consumer package, including dependencies that use these values.
package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"github.com/lucasavila00/immutago/internal/borrow"
	"github.com/lucasavila00/immutago/internal/checker"
	"golang.org/x/tools/go/analysis"
)

// Analyzer checks immutable type obligations and reports unsupported aliasing and mutations.
var Analyzer = &analysis.Analyzer{
	Name:      "immutago",
	Doc:       "prove immutability of marked structs and immutable collection elements",
	Run:       run,
	FactTypes: []analysis.Fact{new(borrow.Summary)},
}

// run checks compiler-resolved types and source operations in one package.
func run(pass *analysis.Pass) (any, error) {
	if pass.Pkg.Path() == checker.LibraryPath {
		return nil, nil
	}
	c := &checks{pass: pass, reported: make(map[token.Pos]bool)}
	for _, file := range pass.Files {
		ast.Inspect(file, c.visit)
	}
	for ident, instance := range pass.TypesInfo.Instances {
		obj := pass.TypesInfo.Uses[ident]
		if obj == nil || obj.Pkg() == nil || obj.Pkg().Path() != checker.LibraryPath {
			continue
		}
		for i := 0; i < instance.TypeArgs.Len(); i++ {
			if reason := checker.Reason(instance.TypeArgs.At(i)); reason != "" {
				c.report(ident.Pos(), "immutable type argument: "+reason)
			}
		}
	}
	if err := borrow.Check(pass); err != nil {
		return nil, err
	}
	return nil, nil
}

// checks carries one package's state and suppresses duplicate reports at the same position.
type checks struct {
	pass     *analysis.Pass
	reported map[token.Pos]bool
}

// report emits a diagnostic once for a source position.
func (c *checks) report(pos token.Pos, message string) {
	if !c.reported[pos] {
		c.reported[pos] = true
		c.pass.Reportf(pos, "%s", message)
	}
}

// visit checks struct contracts, collection instantiations, and mutation sites.
func (c *checks) visit(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.FuncDecl:
		if n.Body == nil {
			c.report(n.Pos(), "unsupported function without a Go body; immutability cannot be proved")
		}
	case *ast.ImportSpec:
		c.checkImport(n)

	case *ast.TypeSpec:
		if obj := c.pass.TypesInfo.Defs[n.Name]; obj != nil {
			c.checkType(n.Pos(), obj.Type())
		}
	case *ast.StructType:
		c.checkStruct(n.Pos(), c.pass.TypesInfo.TypeOf(n))
	}
	if expr, ok := node.(ast.Expr); ok {
		t := c.pass.TypesInfo.TypeOf(expr)
		if t != nil {
			c.checkType(expr.Pos(), t)
		}
	}
	return true
}

// checkImport resolves both quoted forms before rejecting packages outside the supported proof boundary.
func (c *checks) checkImport(spec *ast.ImportSpec) {
	if path, err := strconv.Unquote(spec.Path.Value); err == nil && (path == "unsafe" || path == "reflect") {
		c.report(spec.Pos(), "unsupported "+path+" import; immutability cannot be proved")
	}
}

// checkType checks collection arguments and disallows pointers that could expose mutable aliases.
func (c *checks) checkType(pos token.Pos, t types.Type) {
	if reflectionValue(t) {
		c.report(pos, "unsupported reflection value; immutability cannot be proved")
	}
	if basic, ok := t.Underlying().(*types.Basic); ok && basic.Kind() == types.UnsafePointer {
		c.report(pos, "unsupported unsafe pointer; immutability cannot be proved")
	}
	if checker.IsCollection(t) {
		if reason := checker.Reason(t); reason != "" {
			c.report(pos, "immutable collection: "+reason)
		}
	}
	// Defined types can inherit a struct without a struct literal in this package.
	if _, ok := types.Unalias(t).(*types.Named); ok && !checker.IsCollection(t) {
		if _, ok := t.Underlying().(*types.Struct); ok {
			c.checkStruct(pos, t)
		}
	}
}

// reflectionValue identifies reflection storage even through pointer aliases and recursive pointer types.
func reflectionValue(t types.Type) bool {
	seen := make(map[types.Type]bool)
	for !seen[t] {
		seen[t] = true
		if named, ok := types.Unalias(t).(*types.Named); ok {
			if pkg := named.Obj().Pkg(); pkg != nil && pkg.Path() == "reflect" {
				return true
			}
		}
		pointer, ok := t.Underlying().(*types.Pointer)
		if !ok {
			return false
		}
		t = pointer.Elem()
	}
	return false
}

// checkStruct enforces marker placement and recursively proves each owned field.
func (c *checks) checkStruct(pos token.Pos, t types.Type) {
	if t == nil || checker.IsCollection(t) {
		return
	}
	s, ok := t.Underlying().(*types.Struct)
	if !ok {
		return
	}
	marked := checker.Marked(t)
	for field := range s.Fields() {
		ft := field.Type()
		if checker.IsLibraryType(ft, "Marker") && !field.Embedded() {
			c.report(pos, "im.Marker must be embedded by value")
			return
		}
		if ptr, ok := ft.Underlying().(*types.Pointer); ok && checker.IsLibraryType(ptr.Elem(), "Marker") {
			c.report(pos, "im.Marker must be embedded by value")
			return
		}
		if marked {
			if reason := checker.Reason(ft); reason != "" {
				c.report(pos, "immutable field "+field.Name()+": "+reason)
				return
			}
		} else if checker.Contains(ft) {
			c.report(pos, "unmarked struct contains immutable field "+field.Name()+"; embed im.Marker")
			return
		} else if checker.PossibleStorage(ft) {
			c.report(pos, "unsupported unmarked field "+field.Name()+": unresolved generic storage")
			return
		}
	}
}
