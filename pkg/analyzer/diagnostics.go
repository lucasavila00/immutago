package analyzer

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/types"
	"strings"

	"github.com/lucasavila00/immutago/internal/checker"
	"golang.org/x/tools/go/analysis"
)

// sourceContext shows the rejected source expression and its compiler-resolved type.
func sourceContext(pass *analysis.Pass, node ast.Node) string {
	var source bytes.Buffer
	if err := format.Node(&source, pass.Fset, node); err != nil {
		return ""
	}
	text := strings.Join(strings.Fields(source.String()), " ")
	if len(text) > 180 {
		text = text[:177] + "..."
	}
	context := fmt.Sprintf("\n  source: %s", text)
	if expr, ok := node.(ast.Expr); ok {
		if t := pass.TypesInfo.TypeOf(expr); t != nil {
			context += "\n  type: " + diagnosticType(t)
		}
	}
	return context
}

// diagnosticType uses source package names to keep resolved type context readable.
func diagnosticType(t types.Type) string {
	return types.TypeString(t, func(pkg *types.Package) string { return pkg.Name() })
}

// subsetFixes maps each excluded operation to its supported rewrite.
var subsetFixes = map[string]string{
	"use a service reference only as ref.Borrow().Method(args); " +
		"do not extract the accessor or service pointer": "keep a concrete supervise.Ref[Service] " +
		"and call ref.Borrow().Method(args); " +
		"see docs/reference/service-references.md",
	"Ref requires a service target that embeds supervise.Service": "use a concrete service struct " +
		"that embeds supervise.Service by value",
	"protected operation is outside the supported expression forms": "use a documented concrete value, pointer, " +
		"projection, or builder operation",
	"protected value conversions are unsupported": "construct the destination marked value from copied fields",
	"native builders require fresh local bindings or append to the same handle": "declare one fresh local builder; " +
		"use values = append(values, element) to extend it",
	"Local pointers cannot enter aggregate storage": "keep a Local value in a local binding, or store a Frozen pointer",
	"Local values and pointers cannot be stored in package variables": "keep Local values in local bindings; " +
		"copy the required data into Frozen values before saving it",
	"immutable references require a concrete marked struct target": "allocate a concrete struct that embeds " +
		"im.Frozen or im.Local; keep Local projections by value and put protected arrays in marked structs",
	"returning Local pointers is unsupported": "return a Local value copy, or use im.Frozen " +
		"for objects whose pointers must be shared",
	"global Local pointers are unsupported":        "keep the Local pointer local, or store a Frozen value",
	"containers of Local pointers are unsupported": "keep Local values in local bindings, or store im.Frozen pointers",
	"user generic declarations cannot define Frozen or Local types": "declare a concrete Frozen or Local type " +
		"without type parameters",
	"user generic declarations require ordinary or persistent Frozen contracts": "use a non-generic declaration " +
		"with concrete field and parameter types",
	"user generic instantiations require ordinary or persistent Frozen type arguments": "use a direct non-generic " +
		"function with concrete parameter types",
	"writes to immutable objects are unsupported": "construct a new value and replace the whole variable " +
		"or mutable holder slot; do not write inside immutable records",
	"immutable references require fresh allocation": "use &T{...}, new(T), or new(value); " +
		"do not take the address of an existing variable, field, or element",
	"protected composite literals require a concrete storage type": "use a concrete " +
		"marked field, value array, or checked builder; copy scalar data before storing it in a generic literal",
	"protected operation is outside the supported statement forms": "use a separate " +
		"assignment, direct synchronous call, value return, or documented conditional/loop form",
	"protected iteration requires a direct frozen All call": "use range frozen.All() " +
		"for protected values; do not create protected references through a custom iterator",
	"protected storage requires synchronous calls": "call the function or method directly before " +
		"continuing; keep Local values, Local pointers, and native builders out of go and defer",
	"protected storage is unsupported in type switches": "keep concrete protected " +
		"types in named function parameters instead of recovering values or references from interfaces",
	"immutable methods require a concrete non-generic marked receiver": "declare the method on a proved " +
		"marked struct or its pointer, or use a named function with concrete parameters",
	"promoted immutable methods are unsupported": "name the embedded field explicitly and call a method " +
		"allowed on that field; copy scalar fields to ordinary locals before calling their methods",
	"immutable method receivers must match their declared type exactly": "use an exact pointer or new(value) for a " +
		"pointer receiver; copy a Copied field or element into a local first; " +
		"use (*pointer).Method() for a value receiver",
	"multiple assignments involving protected storage are unsupported": "use one protected destination, " +
		"or local names for all results of one direct typed call",
	"protected storage is unsupported in select communication clauses": "keep Local values and native builders " +
		"out of channel expressions and receive targets; use them in the selected case body",
	"Freeze requires a standalone single binding": "assign frozen := im.FreezeSlice(builder) " +
		"or frozen := im.FreezeMap(builder) in a separate statement before using the frozen value",
	"freeze functions must be called directly": "call im.FreezeMap(builder) or im.FreezeSlice(builder) " +
		"directly so ownership consumption is checked",
	"reference operation is outside the supported allowlist": "use a documented reference form: fresh construction, " +
		"a concrete variable, a read, or a direct typed call; only Frozen pointers can be stored or returned",
	"native slices of immutable storage are unsupported": "use im.FrozenSlice[T] and All() for " +
		"iteration, or build a fresh local slice and consume it with im.FreezeSlice",
	"native maps of immutable storage are unsupported": "use im.FrozenMap[K, V]; pass an " +
		"owned local map or fresh literal to im.FreezeMap",
	"channels of immutable storage are unsupported": "send a Frozen pointer or Frozen handle; " +
		"keep Local data local and do not send Frozen records by value",
	"pointer indirection is unsupported": "use a direct pointer to a concrete marked struct",
	"interior addresses of protected storage are unsupported": "copy the field into a local; " +
		"do not take a pointer into immutable storage",
	"slice projections of immutable storage are unsupported": "copy the array into local " +
		"storage before slicing it, or use im.FrozenSlice for immutable sequence values",
	"shared pointer conversions are unsupported": "keep the original pointer type; " +
		"for a nil Local pointer, declare var p *T = nil",
	"Frozen interface implementations cannot use promoted methods": "convert the explicit embedded Frozen field, " +
		"or declare the method on the converted record",
	"Frozen interface method receivers must match the converted type exactly": "convert a value for value methods " +
		"or a fresh pointer for pointer methods; do not rely on implicit dereferences",
	"Frozen interfaces require concrete Frozen record implementations": "use a proved record that embeds im.Frozen " +
		"or a pointer to that record",
	"ordinary interfaces require ordinary or persistent Frozen values": "keep the concrete immutable " +
		"type in parameters, results, and variables instead of using an interface",
	"interface assertions require ordinary or persistent Frozen values": "keep the concrete immutable " +
		"type in parameters, results, and variables instead of using an interface",
	"dynamic calls require ordinary or persistent Frozen parameters and results": "call a named, non-generic " +
		"function directly with a concrete immutable parameter",
	"callbacks require ordinary or persistent Frozen parameters and results": "call a named, non-generic " +
		"function directly with a concrete immutable parameter",
	"method expressions require ordinary or persistent Frozen receivers and contracts": "call the receiver method " +
		"directly (value.Method(...)); do not call it through the type",
	"method values require ordinary or persistent Frozen receivers and contracts": "call the method directly " +
		"(value.Method(...)); do not save or pass its method value",
	"closures require ordinary or persistent Frozen captures": "copy scalar data before the closure, or use a named " +
		"function with a concrete immutable parameter",
	"immutable iterators must be consumed by direct range": "write for key, value := range " +
		"collection.All() { ... }; do not retain or pass the iterator",
}

// shapeFailure pairs a fixed source rule with its supported replacement.
func shapeFailure(rule string) checker.Failure {
	fix, ok := subsetFixes[rule]
	if !ok {
		fix = "use the documented immutable value and reference forms"
	}
	return checker.Failure{Reason: rule, Repair: fix}
}

// unmarkedFieldFix gives storage replacements without changing the containing type's rules.
func unmarkedFieldFix(t types.Type) string {
	if _, named := types.Unalias(t).(*types.Named); named && checker.Frozen(t) {
		return "store *" + diagnosticType(t) + "; or make the containing struct a Frozen record with Frozen fields"
	}
	for _, kind := range []string{"String", "Slice", "Map"} {
		if checker.IsLibraryType(t, kind+"Projection") {
			return "store the matching im.Frozen" + kind + " value; create its projection at the read site"
		}
	}
	return "store Frozen pointers or handles; keep Local data in local bindings"
}
