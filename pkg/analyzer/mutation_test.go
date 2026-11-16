package analyzer

import (
	"strings"
	"testing"
)

const immutableValueTypes = `type Child struct { im.Frozen; X, Y int }
 type Value struct { im.Frozen; X, Y int; Child Child; Array [2]int }
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
			if !strings.Contains(reports, "interior addresses of protected storage are unsupported") {
				t.Fatalf("scalar address accepted: %s", reports)
			}
		})
	}
}

// TestRecursivePointerType checks that unsupported-type detection terminates on recursive pointers.
func TestRecursivePointerType(t *testing.T) {
	reports := checkSource(t, "type P *P; var _ = im.Frozen{}")
	if reports != "" {
		t.Fatalf("ordinary pointer type rejected: %s", reports)
	}
}

// TestIndirectReferenceKeepsTargetImmutable checks writes after an ordinary pointer-slot read.
func TestIndirectReferenceKeepsTargetImmutable(t *testing.T) {
	source := immutableValueTypes + `
 func dereference(pointer **Value) *Value { return *pointer }
 func f(v Value) { p:=new(v); q:=dereference(&p); q.X=2; _=read(q) }
 `
	reports := checkSource(t, source)
	if !strings.Contains(reports, "writes to immutable objects") {
		t.Fatalf("diagnostics = %q; want immutable target write rejection", reports)
	}
}

// TestUnknownFactoryCannotReturnLocal rejects dynamic Local results.
func TestUnknownFactoryCannotReturnLocal(t *testing.T) {
	source := `type Value struct { im.Local; X int }
 func f(factory func() Value) { value:=factory(); _=value.X }
 `
	reports := checkSource(t, source)
	if !strings.Contains(reports, "dynamic calls require ordinary or Frozen parameters and results") {
		t.Fatalf("diagnostics = %q; want dynamic result rejection", reports)
	}
}
