// Package analyzer exposes the immutago go/analysis analyzer.
//
// The analyzer checks:
//   - Recursive field and collection types for structs with im.Shared or im.Scoped.
//   - Allowed construction, reads, calls, and assignments; no protected field writes.
//   - Shared pointer storage and Scoped pointer retention.
//   - Exclusive native builders, Freeze consumption, and use after Freeze.
//   - No result or one Scoped result for each exported user service method except Init
//     and Run, including promoted user methods. Runtime methods are exempt.
//
// Direct im or supervise imports, used types, and failed checked dependencies
// select packages for analysis. Ordinary Go API contracts are trusted.
// im.UncheckedValue records the declaring package's promise that a value never
// changes. The checker does not prove this promise.
//
// The complete source forms and package rules are in
// https://github.com/lucasavila00/immutago/blob/main/docs/rules.md.
package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lucasavila00/immutago/internal/checker"
	"github.com/lucasavila00/immutago/internal/freeze"
	"golang.org/x/tools/go/analysis"
)

// Analyzer checks immutable type obligations and reports unsupported aliasing and mutations.
var Analyzer = &analysis.Analyzer{
	Name:      "immutago",
	Doc:       "prove immutability of marked structs and immutable collection elements",
	Run:       run,
	FactTypes: []analysis.Fact{new(invalidPackage)},
}

// run checks compiler-resolved types and source operations in one package.
func run(pass *analysis.Pass) (any, error) {
	path := pass.Pkg.Path()
	if path == checker.LibraryPath || path == checker.RuntimePath || testMainPackage(pass.Pkg) || standardLibrary(pass) {
		return nil, nil
	}
	relevant, imports := collectDependencyFacts(pass)
	if !relevant {
		return nil, nil
	}
	immutable := referencesLibrary(pass, false)
	if immutable && rejectCgo(pass, imports) {
		return nil, nil
	}
	reportDependencyFailures(pass, imports)
	invalid := len(imports) != 0
	var failures []proofFailure
	report := pass.Report
	pass.Report = func(d analysis.Diagnostic) {
		invalid = true
		failures = append(failures, localFailure(pass, d))
		report(d)
	}
	defer func() {
		pass.Report = report
		if invalid && pass.ExportPackageFact != nil {
			pass.ExportPackageFact(packageFailureFact(pass.Pkg.Path(), relevant, imports, failures))
		}
	}()
	c := &checks{pass: pass, reported: make(map[token.Pos]bool)}
	for _, file := range pass.Files {
		if immutable {
			ast.Inspect(file, c.visit)
		} else {
			ast.Inspect(file, c.visitService)
		}
	}
	if immutable {
		checkSubset(pass, freeze.Check(pass), &c.proofs)
	}
	return nil, nil
}

// checkLibraryArguments proves im type arguments and protected runtime storage.
func (c *checks) checkLibraryArguments(ident *ast.Ident, instance types.Instance) {
	obj := c.pass.TypesInfo.Uses[ident]
	if runtimeObject(obj) && obj.Name() == "Service" {
		c.checkServiceStorage(ident, instance.TypeArgs.At(0))
		return
	}
	if obj == nil || obj.Pkg() == nil || obj.Pkg().Path() != checker.LibraryPath {
		return
	}
	if obj.Name() == "UncheckedValue" {
		return
	}
	if obj.Name() == "Trust" || obj.Name() == "Unwrap" {
		c.checkTrustedPackage(ident.Pos(), instance.TypeArgs.At(0), "im."+obj.Name())
		return
	}
	if obj.Name() == "Owner" {
		arg := instance.TypeArgs.At(0)
		if !checker.Scoped(arg) || checker.Shared(arg) || c.proofs.Reason(arg) != "" {
			c.report(ident.Pos(), "Owner requires a concrete, valid Scoped struct")
		}
		return
	}
	for i := 0; i < instance.TypeArgs.Len(); i++ {
		if reason := c.proofs.Reason(instance.TypeArgs.At(i)); reason != "" {
			c.report(ident.Pos(), fmt.Sprintf("immutable type argument: %s\n  type: %s\n  fix: %s",
				reason, diagnosticType(instance.TypeArgs.At(i)), structuralFix(instance.TypeArgs.At(i), reason)))
		}
	}
}

// checks carries one package's state and suppresses duplicate reports at the same position.
type checks struct {
	proofs   checker.Prover
	queries  checker.Classifier
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
	case *ast.Ident:
		if instance, ok := c.pass.TypesInfo.Instances[n]; ok {
			c.checkLibraryArguments(n, instance)
		}
	case *ast.FuncDecl:
		if n.Body == nil && usesLibraryType(c.pass.TypesInfo.Defs[n.Name].Type(), make(map[types.Type]bool), false) {
			c.report(n.Pos(), "unsupported function without a Go body; immutability cannot be proved\n  fix: "+
				"provide a Go implementation so writes and pointer transport can be checked")
		}

	case *ast.TypeSpec:
		if obj := c.pass.TypesInfo.Defs[n.Name]; obj != nil {
			c.checkType(n.Pos(), obj.Type())
			c.checkServiceResults(n.Name.Pos(), obj.Type())
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

// checkType proves collection arguments and named struct fields.
func (c *checks) checkType(pos token.Pos, t types.Type) {
	if checker.IsCollection(t) {
		if reason := c.proofs.Reason(t); reason != "" {
			c.report(pos, "immutable collection: "+reason+"\n  type: "+diagnosticType(t)+
				"\n  fix: "+structuralFix(t, reason))
		}
	}
	// Defined types can inherit a struct without a struct literal in this package.
	if _, ok := types.Unalias(t).(*types.Named); ok && !checker.IsCollection(t) {
		if _, ok := t.Underlying().(*types.Struct); ok {
			c.checkStruct(pos, t)
		}
	}
}

// checkStruct enforces marker placement and recursively proves each owned field.
func (c *checks) checkStruct(pos token.Pos, t types.Type) {
	if t == nil || checker.IsCollection(t) || checker.IsLibraryType(t, "Owner") || checker.IsUncheckedValue(t) ||
		checker.IsRuntimeType(t, checker.ServiceMarker) {
		return
	}
	s, ok := t.Underlying().(*types.Struct)
	if !ok {
		return
	}
	marked := checker.Marked(t)
	if checker.Shared(t) && checker.Scoped(t) {
		c.report(pos, "struct cannot embed both im.Shared and im.Scoped")
	} else if checker.MarkerCount(t) > 1 {
		c.report(pos, "struct must embed exactly one immutable marker; each embedded alias counts as a marker")
	}
	for field := range s.Fields() {
		ft := field.Type()
		fieldPos := pos
		if field.Pkg() == c.pass.Pkg && field.Pos().IsValid() {
			fieldPos = field.Pos()
		}
		if invalidMarkerField(field) {
			c.report(fieldPos, "immutable markers must be embedded by value\n  field: "+diagnosticType(t)+"."+field.Name()+
				"\n  fix: embed im.Shared or im.Scoped; remove the marker field name or pointer")
			continue
		}
		if marked {
			if reason := c.proofs.Reason(ft); reason != "" {
				c.report(fieldPos, "immutable field "+field.Name()+": "+reason+
					"\n  owner: "+diagnosticType(t)+"; field type: "+diagnosticType(ft)+
					"\n  fix: "+structuralFix(ft, reason))
				continue
			}
		} else if c.queries.Contains(ft) {
			c.report(fieldPos, "unmarked struct contains immutable field "+field.Name()+"; embed an immutable marker"+
				"\n  owner: "+diagnosticType(t)+"; field type: "+diagnosticType(ft)+
				"\n  rule: a struct that owns immutable storage must itself have an immutable field proof"+
				"\n  fix: embed im.Shared or im.Scoped and make all fields immutable, or remove the immutable field")
			continue
		}
	}
}

// invalidMarkerField rejects named marker fields and pointers to either marker.
func invalidMarkerField(field *types.Var) bool {
	if markerType(field.Type()) {
		return !field.Embedded()
	}
	pointer, ok := field.Type().Underlying().(*types.Pointer)
	return ok && markerType(pointer.Elem())
}

// markerType identifies either exact library marker before its placement is checked.
func markerType(t types.Type) bool {
	return checker.IsLibraryType(t, "Shared") || checker.IsLibraryType(t, "Scoped")
}
