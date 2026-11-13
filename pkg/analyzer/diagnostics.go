package analyzer

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/types"
	"strings"

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
	"immutable references require a concrete marked struct target": "allocate a concrete struct that embeds " +
		"im.Immut or im.ImmutNoRef; copy arrays and frozen collection values without taking their addresses",
	"returning ImmutNoRef references is unsupported": "return an ImmutNoRef value copy, or use im.Immut " +
		"for objects whose references must be shared",
	"global ImmutNoRef references are unsupported":    "keep the ImmutNoRef reference local, or store a value copy",
	"arrays of ImmutNoRef references are unsupported": "store ImmutNoRef value copies or shared im.Immut references",
	"user generic declarations with immutable storage are unsupported": "use a non-generic declaration " +
		"with concrete immutable types; the fixed im collection APIs remain supported",
	"user generic instantiations with immutable types are unsupported": "use a non-generic declaration " +
		"with concrete immutable types; the fixed im collection APIs remain supported",
	"writes to immutable objects are unsupported": "construct a new value, or replace the whole variable; " +
		"only a native builder's whole elements can be changed before Freeze",
	"immutable references require fresh allocation": "use &T{...}, new(T), or new(value); " +
		"do not take the address of an existing variable, field, or element",
	"protected composite literals require a concrete storage type": "use a concrete " +
		"marked field, value array, or checked builder; copy scalar data before storing it in a generic literal",
	"protected operation is outside the supported statement forms": "use a separate " +
		"assignment, direct synchronous call, value return, or documented conditional/loop form",

	"protected iteration requires a direct frozen All call": "use range frozen.All() " +
		"for protected values; do not create protected references through a custom iterator",

	"protected storage requires synchronous calls": "call the reader directly before " +
		"continuing; do not pass protected values, references, or builders to go or defer",
	"protected storage is unsupported in type switches": "keep concrete protected " +
		"types in named function parameters instead of recovering values or references from interfaces",

	"user-defined methods on protected receivers are unsupported": "use a named non-generic " +
		"reader function with a concrete parameter; only direct Get, At, Len, All, and Clone " +
		"calls on im collections are allowed",
	"references require one fixed declaration from an owned local": "declare p := &owner or " +
		"var p *T = &owner once; do not copy, clear, or reassign the reference",
	"multiple assignments involving protected storage are unsupported": "write one protected " +
		"destination per statement; bind a direct frozen.Get result only to two new local variables",
	"protected storage is unsupported in select communication clauses": "keep protected " +
		"values and native builders out of channel expressions and receive targets; use them in the selected case body",
	"Freeze requires a standalone single binding": "assign frozen := im.FreezeSlice(builder) " +
		"or frozen := im.FreezeMap(builder) in a separate statement before using the frozen value",

	"freeze functions must be called directly": "call im.FreezeMap(builder) or im.FreezeSlice(builder) " +
		"directly so ownership consumption is checked",
	"freezing requires a synchronous call": "freeze the local builder directly before " +
		"starting deferred or concurrent work",

	"borrowing through a shared reference is unsupported": "read the child through the " +
		"existing reference, or copy the child into an owned local before borrowing it",
	"reference operation is outside the supported allowlist": "keep the borrow in a local *T " +
		"binding; read its fields, copy *p, compare with nil, or pass it to a direct typed reader",
	"native slices of immutable storage are unsupported": "use im.FrozenSlice[T] and All() for " +
		"iteration, or build a fresh local slice and consume it with im.FreezeSlice",
	"native maps of immutable storage are unsupported": "use im.FrozenMap[K, V]; pass an " +
		"owned local map or fresh literal to im.FreezeMap",
	"channels of immutable storage are unsupported": "keep immutable values in " +
		"owned locals or im collections; channels of these values are outside this subset",
	"pointer-owned construction is unsupported": "construct a local value (v := " +
		"T{...}), then use &v for a read-only borrow",
	"returning shared references is unsupported": "return an owned value T " +
		"instead of *T; copy the pointed-to value before returning",
	"pointer indirection is unsupported": "store owned values, and " +
		"create a direct *T borrow only in the local function that reads it",
	"arrays of shared references are unsupported": "store owned values, and " +
		"create a direct *T borrow only in the local function that reads it",
	"scalar interior addresses are unsupported": "read or update the field " +
		"through its owned value; do not take a field pointer or call its pointer-receiver method",
	"scalar interior pointer methods are unsupported": "read or update the field " +
		"through its owned value; do not take a field pointer or call its pointer-receiver method",
	"slice views of immutable storage are unsupported": "copy the array into local " +
		"storage before slicing it, or use im.FrozenSlice for immutable sequence values",
	"only whole local values can be borrowed": "copy the value into a named " +
		"local, then borrow that local or one of its marked fields",
	"only whole local values and marked fields can be borrowed": "copy the value into a named " +
		"local, then borrow that local or one of its marked fields",
	"shared pointer conversions are unsupported": "keep the original pointer " +
		"type; copy the value first if a different owned value type is required",
	"global shared references are unsupported": "store an owned value " +
		"globally; create and consume shared borrows inside one function",
	"boxing immutable storage in interfaces is unsupported": "keep the concrete immutable " +
		"type in parameters, results, and variables instead of using an interface",
	"immutable interface assertions are unsupported": "keep the concrete immutable " +
		"type in parameters, results, and variables instead of using an interface",
	"dynamic calls with immutable storage are unsupported": "call a named, non-generic " +
		"function directly with a concrete immutable parameter",
	"immutable callbacks are unsupported": "call a named, non-generic " +
		"function directly with a concrete immutable parameter",
	"immutable method expressions are unsupported": "call the method on the collection directly " +
		"(value.Method(...)); do not call it through the collection type",
	"bound immutable methods are unsupported": "call the method directly " +
		"(value.Method(...)); do not save or pass its method value",
	"closures cannot capture immutable storage": "move the code to a named " +
		"function and pass the immutable value or a synchronous read-only borrow explicitly",
	"generic calls with immutable types are unsupported": "use a non-generic function " +
		"with concrete immutable parameter types; only the fixed im APIs support these calls",
	"generic borrowing calls are unsupported": "use a non-generic function " +
		"with concrete immutable parameter types; only the fixed im APIs support these calls",
	"shared references require synchronous calls": "make the read-only call " +
		"directly; do not use go or defer with this borrow",
	"shared references cannot enter aggregate storage": "store an owned value copy " +
		"instead of a pointer; keep shared references in local pointer variables",
	"shared references cannot enter containers": "store an owned value copy " +
		"instead of a pointer; keep shared references in local pointer variables",
	"immutable iterators must be consumed by direct range": "write for key, value := range " +
		"collection.All() { ... }; do not retain or pass the iterator",
}

// subsetFix gives a specific supported rewrite for each excluded borrow transport.
func subsetFix(rule string) string {
	if fix, ok := subsetFixes[rule]; ok {
		return fix
	}
	return "use owned value copies and direct, synchronous, read-only borrows within one function"
}

// structuralFix explains how to replace a field or type argument that failed its recursive proof.
func structuralFix(t types.Type, reason string) string {
	switch {
	case strings.Contains(reason, "named array"):
		return "use an unnamed array type, or wrap the named array in a struct that embeds im.Marker"
	case strings.Contains(reason, "constraint"):
		return "restrict the type parameter to immutable concrete types; any and mutable alternatives cannot prove this field"
	case strings.Contains(reason, "does not embed"):
		return "embed im.Marker in the owned struct and make each of its fields immutable"
	}
	switch t.Underlying().(type) {
	case *types.Pointer:
		return "replace the pointer field or argument with an owned immutable value"
	case *types.Slice:
		return "replace the native slice with im.FrozenSlice[T], where T is immutable"
	case *types.Map:
		return "replace the native map with im.FrozenMap[K, V], where K and V are immutable"
	case *types.Interface:
		return "replace the interface with a concrete immutable type"
	case *types.Chan, *types.Signature:
		return "keep channels and callbacks outside immutable values; store their immutable results"
	}
	return "use primitive values, unnamed arrays, marked structs, or " +
		"im.FrozenSlice/im.FrozenMap with immutable type arguments"
}
