package analyzer

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
)

const immutableValueTypes = `type Child struct { im.Shared; X, Y int }
 type Value struct { im.Shared; X, Y int; Child Child; Array [2]int }
 func read(pointer *Value) int { return pointer.X }
 func readInt(pointer *int) int { return *pointer }
 `

// TestImmutableConstruction accepts scalar preparation and complete value replacement.
func TestImmutableConstruction(t *testing.T) {
	cases := []struct{ name, source string }{
		{"conditional", `func f(condition bool)Value{x:=2;if condition{x=1};return Value{X:x}}`},
		{"value replacement", `func f(v Value)Value{v=Value{X:1};v=Value{X:2};return v}`},
		{"array", `func f()Value{return Value{Array:[2]int{1,2}}}`},
		{"range", `func f(v Value)Value{for i:=range 2{v=Value{X:i}};return v}`},
		{"value function", `func WithX(v Value,x int)Value{return Value{X:x,Y:v.Y,Child:v.Child,Array:v.Array}}`},
		{"read pointer", `func f(v *Value)int{return v.X}`},
		{"copy", `func f(v Value)Value{p:=new(v);copy:=v;copy=Value{X:2};_=read(p);return copy}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if reports := checkSource(t, immutableValueTypes+tc.source); reports != "" {
				t.Fatalf("unexpected diagnostics: %s", reports)
			}
		})
	}
}

// TestImmutableBindingReplacement checks replacement across ordinary branches and loops.
func TestImmutableBindingReplacement(t *testing.T) {
	cases := []struct{ name, source string }{
		{"live reference", `func f(v Value){p:=new(v);v=Value{X:2};_=read(p)}`},
		{"copied reference", `func f(v Value){p:=new(v);q:=p;p=new(Value{X:2});_=read(q);_=read(p)}`},
		{"branch", `func f(v Value,b bool){p:=new(v);if b{p=&Value{X:1}};_=read(p)}`},
		{"branch join", `func f(a,b Value,c bool){var p *Value;if c{p=new(a)}else{p=new(b)};_=read(p)}`},
		{"loop", `func f(n int){p:=&Value{};for i:=0;i<n;i++{old:=p;p=&Value{X:i};_=read(old)};_=read(p)}`},
		{"early return", `func f(v Value,b bool){p:=new(v);if b{_=read(p);return};v=Value{X:2};_=v}`},
		{"rhs", `func f(v Value){v=Value{X:read(new(v))+1};_=v}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if reports := checkSource(t, immutableValueTypes+tc.source); reports != "" {
				t.Fatalf("unexpected diagnostics: %s", reports)
			}
		})
	}
}

// TestImmutableWrites rejects field and target writes without a lifetime exception.
func TestImmutableWrites(t *testing.T) {
	cases := []struct{ name, source string }{
		{"before sharing", `func f(){v:=Value{};v.X=1}`},
		{"after read", `func f(){p:=&Value{};_=read(p);p.X=2}`},
		{"value copy", `func f(v Value){copy:=v;copy.X=2}`},
		{"fresh pointer", `func f(){p:=&Value{};p.X=1}`},
		{"new pointer", `func f(){p:=new(Value);p.X=1}`},
		{"copy constructor", `func f(v Value){p:=new(v);p.X=1}`},
		{"pointer alias", `func f(){p:=&Value{};q:=p;p.X=2;_=read(q)}`},
		{"cleared alias", `func f(){p:=&Value{};q:=p;p=nil;q.X=2;_=p}`},
		{"field", `func f(v Value){v.X=2}`},
		{"nested field", `func f(v Value){v.Child.X=2}`},
		{"nested value", `func f(v Value){v.Child=Child{}}`},
		{"array element", `func f(v Value){v.Array[0]=2}`},
		{"dynamic array element", `func f(v Value,i int){v.Array[i]=2}`},
		{"compound", `func f(v Value){v.X+=2}`},
		{"increment", `func f(v Value){v.X++}`},
		{"range", `func f(v Value){for v.X=range 2{}}`},
		{"branch", `func f(v Value,b bool){if b{v.X=1}else{v.X=2}}`},
		{"loop", `func f(v Value,n int){for i:=0;i<n;i++{v.X=i}}`},
		{"builder value", `func f(){values:=[]Value{{}};values[0].X=2}`},
		{"builder reference", `func f(){values:=[]*Value{&Value{}};values[0].X=2}`},
		{"pointer target", `func f(v *Value){*v=Value{}}`},
		{"package value", `var Global Value;func f(){Global.X=2}`},
		{"package reference", `var Global=&Value{};func f(){Global.X=2}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reports := checkSource(t, immutableValueTypes+tc.source)
			if !strings.Contains(reports, "writes to immutable objects") {
				t.Fatalf("immutable write accepted: %s", reports)
			}
		})
	}
}

// TestScalarAddressesCannotLeaveImmutableStorage checks scalar aliases in ordinary aggregate storage.
func TestScalarAddressesCannotLeaveImmutableStorage(t *testing.T) {
	cases := []string{
		`func f(v Value){h:=struct{P *int}{P:&v.X};*h.P=2}`,
		`func f(v Value){h:=[]*int{&v.X};*h[0]=2}`,
		`func f(v Value){h:=map[string]*int{"x":&v.X};*h["x"]=2}`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			reports := checkSource(t, immutableValueTypes+source)
			if !strings.Contains(reports, "scalar interior addresses are unsupported") {
				t.Fatalf("scalar address accepted: %s", reports)
			}
		})
	}
}

// TestUnsafeHasNoSuppression checks that source comments do not disable the unsafe import boundary.
func TestUnsafeHasNoSuppression(t *testing.T) {
	source := `import "unsafe"
 //nolint:immutago
 var _ = unsafe.Sizeof(im.Shared{})`
	if reports := checkSource(t, source); !strings.Contains(reports, "unsupported unsafe import") {
		t.Fatalf("diagnostics = %q; want unsafe diagnostic", reports)
	}
}

// TestUnsafeReflection checks unsafe pointers returned without an explicit unsafe import.
func TestUnsafeReflection(t *testing.T) {
	source := `import "reflect"
 var _ = im.Shared{}
 func pointer(v reflect.Value) { _ = v.UnsafePointer() }`
	if reports := checkSource(t, source); !strings.Contains(reports, "unsupported unsafe pointer") {
		t.Fatalf("diagnostics = %q; want unsafe pointer diagnostic", reports)
	}
}

// TestRawUnsafeImport checks that alternate import quoting cannot bypass the unsafe boundary.
func TestRawUnsafeImport(t *testing.T) {
	source := "import `unsafe`\nvar _ = unsafe.Sizeof(im.Shared{})"
	if reports := checkSource(t, source); !strings.Contains(reports, "unsupported unsafe import") {
		t.Fatalf("diagnostics = %q; want unsafe import diagnostic", reports)
	}
}

// TestReflectionMutation checks the safe-Go reflection path into a marked slice element.
func TestReflectionMutation(t *testing.T) {
	source := "import `reflect`\n" + `type V struct { im.Shared; N int }
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
	reports := checkSource(t, "type P *P; var _ = im.Shared{}")
	if reports != "" {
		t.Fatalf("ordinary pointer type rejected: %s", reports)
	}
}

// TestIndirectReferenceCannotErasePermission rejects access through a pointer-variable address.
func TestIndirectReferenceCannotErasePermission(t *testing.T) {
	source := immutableValueTypes + `
 func dereference(pointer **Value) *Value { return *pointer }
 func f(v Value) { p:=new(v); q:=dereference(&p); v.X=2; _=read(q) }
 `
	reports := checkSource(t, source)
	if !strings.Contains(reports, "pointer indirection is unsupported") {
		t.Fatalf("diagnostics = %q; want unsupported pointer indirection", reports)
	}
}

// TestUnknownFactoryCannotReturnProtectedStorage rejects dynamic protected results.
func TestUnknownFactoryCannotReturnProtectedStorage(t *testing.T) {
	source := immutableValueTypes + `
 func f(factory func() *Value) { pointer:=factory(); _=pointer.X }
 `
	reports := checkSource(t, source)
	if !strings.Contains(reports, "dynamic calls with immutable storage are unsupported") {
		t.Fatalf("diagnostics = %q; want dynamic result rejection", reports)
	}
}
