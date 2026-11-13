package analyzer

import (
	"go/token"
	"go/types"
	"golang.org/x/tools/go/analysis"
	"strings"
	"testing"
)

const borrowTestTypes = `type Child struct { im.Marker; X, Y int }
 type Value struct { im.Marker; X, Y int; Child Child; Array [2]int }
 func read(pointer *Value) int { return pointer.X }
 func readInt(pointer *int) int { return *pointer }
 func (value *Value) Read() int { return value.X }
 `

// TestOwnedConstruction accepts changes to owned values and copies without a live shared loan.
func TestOwnedConstruction(t *testing.T) {
	cases := []struct{ name, source string }{
		{"conditional", `func f(condition bool) Value { var v Value; if condition {v.X=1} else {v.X=2}; return v }`},
		{"repeated assignment", `func f(v Value) Value { v.X=1; v.X=2; v.X++; v.Y+=3; return v }`},
		{"array", `func f(v Value) Value { v.Array[0]=1; return v }`},
		{"range", `func f(v Value) Value { for v.X=range 2 {}; for v.Array[0]=range 2 {}; return v }`},
		{"value receiver", `func (v Value) WithX(x int) Value { v.X=x; return v }`},
		{"read pointer", `func f(v *Value) int { return v.X }`},
		{"copy", `func f(v Value) Value { p:=&v; copy:=v; copy.X=2; _=read(p); return copy }`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if reports := checkSource(t, borrowTestTypes+tc.source); reports != "" {
				t.Fatalf("unexpected diagnostics:\n%s", reports)
			}
		})
	}
}

// TestBorrowLifetimes distinguishes final-use liveness from lexical scope and permanent freezing.
func TestBorrowLifetimes(t *testing.T) {
	cases := []struct{ name, source string }{
		{"last use", `func f(v Value) { p:=&v; _=read(p); v.X=2; v=Value{} }`},
		{"copy last use", `func f(v Value) { p:=&v; q:=p; p=nil; _=read(q); v.X=2; _=p }`},
		{"rebind", `func f(a,b Value) { p:=&a; _=read(p); p=&b; a.X=2; _=read(p); b.X=2 }`},
		{"branch", `func f(v Value,b bool) { if b {p:=&v; _=read(p)} else {v.X=1}; v.X=2 }`},
		{"branch join", `func f(a,b Value,c bool) { var p *Value; if c {p=&a} else {p=&b}; _=read(p); a.X=1; b.X=2 }`},
		{"loop", `func f(v Value,n int) { for i:=0;i<n;i++ {v.X=i; p:=&v; _=read(p)}; v.X=2 }`},
		{"loop alias last use", `func f(v Value,n int) { var old *Value
         for i:=0;i<n;i++ {if old!=nil {_=read(old)}; v.X=i; p:=&v; old=p} }`},
		{"early return", `func f(v Value,b bool) { p:=&v; if b {_=read(p);return}; v.X=2 }`},
		{"rhs", `func f(v Value) { v.X=read(&v)+1 }`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if reports := checkSource(t, borrowTestTypes+tc.source); reports != "" {
				t.Fatalf("unexpected diagnostics:\n%s", reports)
			}
		})
	}
}

// TestLiveBorrowConflicts rejects writes that overlap a shared loan or use a read-only reference.
func TestLiveBorrowConflicts(t *testing.T) {
	cases := []struct{ name, source string }{
		{"collection native copy", `func f() { values:=im.FreezeSlice([]Value{{}}).Clone()
         p:=&values[0]; values[0].X=2; _=read(p) }`},
		{"slice view", `func f(v Value) { view:=v.Array[:]; v.Array[0]=2; _=view[0] }`},
		{"slice view write", `func f(v Value) { view:=v.Array[:]; view[0]=2 }`},
		{"owner live alias", `func f() { p:=&Value{}; q:=p; p.X=2; _=read(q) }`},
		{"shared owner alias", `func f() { p:=&Value{}; q:=p; q.X=2 }`},
		{"scalar mutator", `func set(p *int) {*p=2}; func f(v Value) { set(&v.X) }`},
		{"whole owner field", `func f(v Value) { p:=&v.Child; v.X=2; _=p.X }`},
		{"fresh pointer write", `func f() {p:=&Value{};p.X=1}`},
		{"new pointer write", `func f() {p:=new(Value);p.X=1}`},
		{"field", `func f(v Value) { p:=&v; v.X=2; _=read(p) }`},
		{"whole", `func f(v Value) { p:=&v; v=Value{}; _=read(p) }`},
		{"copied alias", `func f(v Value) { p:=&v; q:=p; p=nil; v.X=2; _=read(q); _=p }`},
		{"nested parent", `func f(v Value) { p:=&v.Child.X; v.Child=Child{}; _=readInt(p) }`},
		{"array", `func f(v Value) { p:=&v.Array[0]; v.Array[0]=2; _=readInt(p) }`},
		{"dynamic array", `func f(v Value,i,j int) { p:=&v.Array[i]; v.Array[j]=2; _=readInt(p) }`},
		{"branch", `func f(a,b Value,c bool) { var p *Value; if c {p=&a} else {p=&b}; a.X=2; _=read(p) }`},
		{"loop", `func f(v Value,n int) { p:=&v; for i:=0;i<n;i++ {v.X=i; _=read(p)} }`},
		{"loop loan reissuance", `func f(v Value,n int) { var old *Value
         for i:=0;i<n;i++ {v.X=i; p:=&v; if old!=nil {_=read(old)}; old=p} }`},
		{"identity", `func identity[T any](p *T)*T{return p}; func f(v Value) { p:=identity(&v); v.X=2; _=read(p) }`},
		{"method value", `func f(v Value) { later:=v.Read; v.X=2; _=later() }`},
		{"closure", `func f(v Value) { p:=&v; later:=func()int{return p.X}; v.X=2; _=later() }`},
		{"interface", `func f(v Value) { var boxed any=&v; v.X=2; _=read(boxed.(*Value)) }`},
		{"defer", `func f(v Value) { defer read(&v); v.X=2 }`},
		{"assignment order", `func f(v Value) { var p *Value; p,v.X=&v,2; _=read(p) }`},
		{"slice parameter write", `func f(v []Value) { v[0].X=2 }`},
		{"pointer write", `func f(v *Value) { v.X=2 }`},
		{"pointer replacement", `func f(v *Value) { *v=Value{} }`},
		{"scalar interface write", `func f(v Value) { var boxed any=&v.X; *boxed.(*int)=2 }`},
		{"field pointer write", `func f(v Value) { p:=&v.X; *p=2 }`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reports := checkSource(t, borrowTestTypes+tc.source)
			if !strings.Contains(reports, "borrow") && !strings.Contains(reports, "shared reference") &&
				!strings.Contains(reports, "unsupported") {
				t.Fatalf("diagnostics = %q; want borrowed-storage or shared-reference conflict", reports)
			}
		})
	}
}

// TestBorrowCarriersCannotErasePermissions checks scalar aliases retained in ordinary aggregate storage.
func TestBorrowCarriersCannotErasePermissions(t *testing.T) {
	cases := []string{
		`func f(v Value) { h:=struct{ P *int }{P:&v.X}; *h.P=2 }`,
		`func f(v Value) { h:=[]*int{&v.X}; *h[0]=2 }`,
		`func f(v Value) { h:=map[string]*int{"x":&v.X}; *h["x"]=2 }`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			reports := checkSource(t, borrowTestTypes+source)
			if !strings.Contains(reports, "borrow") && !strings.Contains(reports, "shared reference") &&
				!strings.Contains(reports, "unsupported") {
				t.Fatalf("diagnostics = %q; want capability conflict or unsupported retention", reports)
			}
		})
	}
}

// TestUnsafeHasNoSuppression checks that source comments do not disable the unsafe import boundary.
func TestUnsafeHasNoSuppression(t *testing.T) {
	source := `import "unsafe"
 //nolint:immutago
 var _ = unsafe.Sizeof(im.Marker{})`
	if reports := checkSource(t, source); !strings.Contains(reports, "unsupported unsafe import") {
		t.Fatalf("diagnostics = %q; want unsafe diagnostic", reports)
	}
}

// TestUnsafeReflection checks unsafe pointers returned without an explicit unsafe import.
func TestUnsafeReflection(t *testing.T) {
	source := `import "reflect"
 var _ = im.Marker{}
 func pointer(v reflect.Value) { _ = v.UnsafePointer() }`
	if reports := checkSource(t, source); !strings.Contains(reports, "unsupported unsafe pointer") {
		t.Fatalf("diagnostics = %q; want unsafe pointer diagnostic", reports)
	}
}

// TestRawUnsafeImport checks that alternate import quoting cannot bypass the unsafe boundary.
func TestRawUnsafeImport(t *testing.T) {
	source := "import `unsafe`\nvar _ = unsafe.Sizeof(im.Marker{})"
	if reports := checkSource(t, source); !strings.Contains(reports, "unsupported unsafe import") {
		t.Fatalf("diagnostics = %q; want unsafe import diagnostic", reports)
	}
}

// TestReflectionMutation checks the safe-Go reflection path into a marked slice element.
func TestReflectionMutation(t *testing.T) {
	source := "import `reflect`\n" + `type V struct { im.Marker; N int }
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
	reports := checkSource(t, "type P *P; var _ = im.Marker{}")
	if reports != "" {
		t.Fatalf("ordinary pointer type rejected: %s", reports)
	}
}

// TestIndirectBorrowCannotErasePermission checks a shared alias loaded through a pointer-variable address.
func TestIndirectBorrowCannotErasePermission(t *testing.T) {
	source := borrowTestTypes + `
 func dereference(pointer **Value) *Value { return *pointer }
 func f(v Value) { p:=&v; q:=dereference(&p); v.X=2; _=read(q) }
 `
	reports := checkSource(t, source)
	if !strings.Contains(reports, "borrow") && !strings.Contains(reports, "shared reference") &&
		!strings.Contains(reports, "unsupported") {
		t.Fatalf("diagnostics = %q; want an indirect loan conflict or unsupported dereference", reports)
	}
}

// TestUnknownFactoryCannotCreateOwnership rejects protected pointer results with no inferred effects.
func TestUnknownFactoryCannotCreateOwnership(t *testing.T) {
	source := borrowTestTypes + `
 func f(factory func() *Value) { pointer:=factory(); pointer.X=2 }
 `
	reports := checkSource(t, source)
	if !strings.Contains(reports, "unsupported") && !strings.Contains(reports, "shared reference") {
		t.Fatalf("diagnostics = %q; want unknown result ownership rejection", reports)
	}
}
