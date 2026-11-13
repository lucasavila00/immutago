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
	"borrowing through a shared reference is unsupported": "read the child through the " +
		"existing reference, or copy the child into an owned local before borrowing it",
	"reference operation is outside the supported allowlist": "keep the borrow in a local *T " +
		"binding; read its fields, copy *p, compare with nil, or pass it to a direct typed reader",
	"native slices of immutable storage are unsupported": "use im.Slice[T] and All() for " +
		"iteration; pass an inline []T literal only to im.NewSlice",
	"native maps of immutable storage are unsupported": "use im.Map[K, V]; pass an " +
		"inline map literal only to im.NewMap",
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
		"storage before slicing it, or use im.Slice for immutable sequence values",
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
		return "replace the native slice with im.Slice[T], where T is immutable"
	case *types.Map:
		return "replace the native map with im.Map[K, V], where K and V are immutable"
	case *types.Interface:
		return "replace the interface with a concrete immutable type"
	case *types.Chan, *types.Signature:
		return "keep channels and callbacks outside immutable values; store their immutable results"
	}
	return "use primitive values, unnamed arrays, marked structs, or im.Slice/im.Map with immutable type arguments"
}
