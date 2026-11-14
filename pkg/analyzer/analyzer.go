// Package analyzer exposes the immutago go/analysis analyzer.
//
// Structs that embed im.Shared or im.Scoped are immutable from construction.
// Their fields must be scalars, unnamed arrays, other marked structs, pointers
// to concrete Shared structs, or frozen collections of allowed types. Each
// component is checked recursively. Unmarked structs cannot store protected
// values. Interfaces, callbacks, user generics with protected storage,
// reflection, and unsafe operations are unsupported in checked packages.
//
// Whole variable bindings can be replaced. Field, element, and pointer-target
// writes to protected objects are rejected. Protected pointers start with
// fresh construction and target concrete marked structs. Shared pointers can
// be copied, stored, and returned. Scoped pointers stay local or pass to
// direct typed calls that do not retain them. Neither mode permits protected
// closure captures or protected operations in go or defer.
//
// Struct and array copies copy their values; frozen fields share read-only storage.
// Freeze calls consume exclusive native map and slice builders without copying.
// Builder aliases and later uses are rejected; Clone creates independent native storage.
// Direct im or proc imports, locally used im or proc type contracts, and failed
// checked dependencies select proof checking. Other packages use ordinary Go API
// trust, including string immutability. Checked failures propagate to consumers.
//
// A struct that embeds proc.Process[A] is a mutable process. Its pointer is valid
// only as a method receiver or as a direct PID.Proc result that receives a direct
// call of an exported method. Exported process methods take and return only
// allowed stored types, and only proc.Start and proc.StartReserved create processes.
package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

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
	if rejectCgo(pass, imports) {
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
		ast.Inspect(file, c.visit)
	}
	checkLibraryArguments(pass, c)
	checkSubset(pass, freeze.Check(pass))
	return nil, nil
}

// checkLibraryArguments proves type arguments of the fixed library types.
func checkLibraryArguments(pass *analysis.Pass, c *checks) {
	for ident, instance := range pass.TypesInfo.Instances {
		obj := pass.TypesInfo.Uses[ident]
		c.checkRuntimeArguments(ident, obj, instance)
		if obj == nil || obj.Pkg() == nil || obj.Pkg().Path() != checker.LibraryPath {
			continue
		}
		if obj.Name() == "Opaque" {
			continue
		}
		if obj.Name() == "Seal" {
			c.checkTrustedPackage(ident.Pos(), instance.TypeArgs.At(0), "im.Seal")
			continue
		}
		if obj.Name() == "Owner" {
			arg := instance.TypeArgs.At(0)
			if !checker.Scoped(arg) || checker.Shared(arg) || c.proofs.Reason(arg) != "" {
				c.report(ident.Pos(), "Owner requires a concrete, valid Scoped struct")
			}
			continue
		}
		for i := 0; i < instance.TypeArgs.Len(); i++ {
			if reason := c.proofs.Reason(instance.TypeArgs.At(i)); reason != "" {
				c.report(ident.Pos(), fmt.Sprintf("immutable type argument: %s\n  type: %s\n  fix: %s",
					reason, diagnosticType(instance.TypeArgs.At(i)), structuralFix(instance.TypeArgs.At(i), reason)))
			}
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
	case *ast.FuncDecl:
		if n.Body == nil {
			c.report(n.Pos(), "unsupported function without a Go body; immutability cannot be proved\n  fix: "+
				"provide a Go implementation so writes and pointer transport can be checked")
		}
		c.checkProcessMethod(n)
	case *ast.ImportSpec:
		c.checkImport(n)

	case *ast.TypeSpec:
		if obj := c.pass.TypesInfo.Defs[n.Name]; obj != nil {
			c.checkType(n.Pos(), obj.Type())
		}
	case *ast.StructType:
		c.checkStruct(n.Pos(), c.pass.TypesInfo.TypeOf(n))
	case *ast.SelectorExpr:
		c.checkOpaqueValue(n)
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
		c.report(spec.Pos(), "unsupported "+path+" import; immutability cannot be proved\n  rule: reflection and unsafe "+
			"operations can bypass typed ownership checks\n  fix: replace these operations with ordinary typed Go code")
	}
}

// checkType checks collection arguments and disallows pointers that could expose mutable aliases.
func (c *checks) checkType(pos token.Pos, t types.Type) {
	if reflectionValue(t) {
		c.report(pos, "unsupported reflection value; immutability cannot be proved\n  type: "+diagnosticType(t)+
			"\n  fix: replace reflection values with concrete typed values and direct "+
			"operations")
	}
	if basic, ok := t.Underlying().(*types.Basic); ok && basic.Kind() == types.UnsafePointer {
		c.report(pos, "unsupported unsafe pointer; immutability cannot be proved\n  type: "+diagnosticType(t)+
			"\n  fix: use typed value copies or immutable references; do not reinterpret "+
			"storage")
	}
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
	if t == nil || checker.IsCollection(t) || checker.IsLibraryType(t, "Owner") || checker.IsOpaque(t) ||
		checker.IsRuntimeType(t, checker.ProcessMarker) {
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
		if c.unprovedField(fieldPos, t, field, marked) {
			continue
		}
		if marked {
			if reason := c.proofs.Reason(ft); reason != "" {
				c.report(fieldPos, "immutable field "+field.Name()+": "+reason+
					"\n  owner: "+diagnosticType(t)+"; field type: "+diagnosticType(ft)+
					"\n  fix: "+structuralFix(ft, reason))
				continue
			}
		} else if c.queries.Contains(ft) && checker.Process(t) {
			c.report(fieldPos, "process field "+field.Name()+" contains immutable storage; publish it through an im.Owner field")
			continue
		} else if c.queries.Contains(ft) {
			c.report(fieldPos, "unmarked struct contains immutable field "+field.Name()+"; embed an immutable marker"+
				"\n  owner: "+diagnosticType(t)+"; field type: "+diagnosticType(ft)+
				"\n  rule: a struct that owns immutable storage must itself have an immutable field proof"+
				"\n  fix: embed im.Shared or im.Scoped and make all fields immutable, or remove the immutable field")
			continue
		}
	}
}

// unprovedField reports a misplaced marker or proc.Dependency field and identifies a field without a type proof.
func (c *checks) unprovedField(pos token.Pos, t types.Type, field *types.Var, marked bool) bool {
	if invalidMarkerField(field) {
		c.report(pos, "immutable markers must be embedded by value\n  field: "+diagnosticType(t)+"."+field.Name()+
			"\n  fix: embed im.Shared or im.Scoped; remove the marker field name or pointer")
		return true
	}
	if !checker.TrustedField(field) {
		return false
	}
	if !marked {
		c.report(pos, "proc.Dependency field "+field.Name()+" is allowed only in a marked process argument struct"+
			"\n  owner: "+diagnosticType(t)+"\n  fix: move the field into the argument struct of the process")
	}
	return true
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
