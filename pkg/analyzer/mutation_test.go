package analyzer

import (
	"go/token"
	"go/types"
	"golang.org/x/tools/go/analysis"
	"strings"
	"testing"
)

// TestMutationBoundaries checks explicit writes and implicit address operations on immutable fields.
func TestMutationBoundaries(t *testing.T) {
	const prefix = `type Count int
 func (p *Count) Change() { *p = 2 }
 type Value struct { immutable.Marker; N Count; A [2]int }
 `
	cases := []struct{ name, source, want string }{
		{"assignment", `func f(v Value) { v.N = 1 }`, "cannot mutate a field"},
		{"increment", `func f(v Value) { v.N++ }`, "cannot mutate a field"},
		{"compound", `func f(v Value) { v.N += 1 }`, "cannot mutate a field"},
		{"array", `func f(v Value) { v.A[0] = 1 }`, "cannot mutate a field"},
		{"address field", `func f(v Value) { _ = &v.N }`, "cannot mutate a field"},
		{"address element", `func f(v Value) { _ = &v.A[0] }`, "cannot mutate a field"},
		{"slice view", `func f(v Value) { _ = v.A[:] }`, "cannot mutate a field"},
		{"implicit address", `func f(v Value) { v.N.Change() }`, "cannot mutate a field"},
		{"bound method", `func f(v Value) { _ = v.N.Change }`, "cannot mutate a field"},
		{"pointer", `func f(v *Value) {}`, "unsupported pointer"},
		{"address value", `func f(v Value) { _ = &v }`, "unsupported pointer"},
		{"range", `func f(v Value) { for v.N = range 2 {} }`, "cannot mutate a field"},
		{"element range", `func f(v Value) { for v.A[0] = range 2 {} }`, "cannot mutate a field"},
		{"slice element", `func f(v []Value) { v[0].N++ }`, "cannot mutate a field"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if reports := checkSource(t, prefix+tc.source); !strings.Contains(reports, tc.want) {
				t.Fatalf("diagnostics = %q; want %q", reports, tc.want)
			}
		})
	}
}

// TestUnsafeHasNoSuppression checks that source comments do not disable the unsafe import boundary.
func TestUnsafeHasNoSuppression(t *testing.T) {
	source := `import "unsafe"
 //nolint:immutago
 var _ = unsafe.Sizeof(immutable.Marker{})`
	if reports := checkSource(t, source); !strings.Contains(reports, "unsupported unsafe import") {
		t.Fatalf("diagnostics = %q; want unsafe diagnostic", reports)
	}
}

// TestUnsafeReflection checks unsafe pointers returned without an explicit unsafe import.
func TestUnsafeReflection(t *testing.T) {
	source := `import "reflect"
 var _ = immutable.Marker{}
 func pointer(v reflect.Value) { _ = v.UnsafePointer() }`
	if reports := checkSource(t, source); !strings.Contains(reports, "unsupported unsafe pointer") {
		t.Fatalf("diagnostics = %q; want unsafe pointer diagnostic", reports)
	}
}

// TestRawUnsafeImport checks that alternate import quoting cannot bypass the unsafe boundary.
func TestRawUnsafeImport(t *testing.T) {
	source := "import `unsafe`\nvar _ = unsafe.Sizeof(immutable.Marker{})"
	if reports := checkSource(t, source); !strings.Contains(reports, "unsupported unsafe import") {
		t.Fatalf("diagnostics = %q; want unsafe import diagnostic", reports)
	}
}

// TestReflectionMutation checks the safe-Go reflection path into a marked slice element.
func TestReflectionMutation(t *testing.T) {
	source := "import `reflect`\n" + `type V struct { immutable.Marker; N int }
 func f(values []V) { reflect.ValueOf(values).Index(0).FieldByName("N").SetInt(1) }`
	if reports := checkSource(t, source); !strings.Contains(reports, "unsupported reflect import") {
		t.Fatalf("diagnostics = %q; want reflection diagnostic", reports)
	}
}

// TestIndirectReflectionType checks reflection values obtained without an import in the consumer.
func TestIndirectReflectionType(t *testing.T) {
	value := types.NewNamed(types.NewTypeName(token.NoPos, types.NewPackage("reflect", "reflect"), "Value", nil),
		types.NewStruct(nil, nil), nil)
	var message string
	c := &checks{reported: make(map[token.Pos]bool), pass: &analysis.Pass{
		Report: func(d analysis.Diagnostic) { message = d.Message },
	}}
	c.checkType(token.Pos(1), value)
	c.checkType(token.Pos(2), types.NewPointer(value))
	if !strings.Contains(message, "unsupported reflection value") {
		t.Fatalf("diagnostic = %q", message)
	}
}

// TestRecursivePointerType checks that unsupported-type detection terminates on recursive pointers.
func TestRecursivePointerType(t *testing.T) {
	reports := checkSource(t, "type P *P; var _ = immutable.Marker{}")
	if reports != "" {
		t.Fatalf("ordinary pointer type rejected: %s", reports)
	}
}
