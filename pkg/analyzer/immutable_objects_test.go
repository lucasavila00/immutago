package analyzer

import (
	"strings"
	"testing"
)

const immutableTestTypes = `
type Frozen struct { im.Frozen; X int; Array [2]int }
type Local struct { im.Local; X int }
func readSnapshot(value *Frozen) int { if value == nil { return 0 }; return value.X }
func readLocal(value *Local) int { if value == nil { return 0 }; return value.X }
`

// TestImmutableObjectForms checks fresh construction, binding replacement, and the two reference modes.
func TestImmutableObjectForms(t *testing.T) {
	cases := []string{
		`func f(){p:=&Frozen{};_=func()int{return p.X}}`,
		`func f(){p:=&Frozen{};go readSnapshot(p)}`,
		`func f(p *Frozen,boxed any)bool{return p==boxed}`,
		`type Reader int;func(Reader)Read(p *Frozen)int{return p.X};func f(){p:=&Frozen{};_=Reader(0).Read(p)}`,
		`type Reader int;func(Reader)Make()*Frozen{return &Frozen{}};func f(){_=Reader(0).Make()}`,

		`func f()*Frozen { p:=&Frozen{X:1}; q:=p; p=&Frozen{X:2}; _=p; return q }`,
		`var global *Frozen;func f(){global=&Frozen{X:1};global=nil}`,
		`var global Frozen;func f(){global=Frozen{X:1}}`,
		`func f(){p:=new(Frozen);q:=new(Frozen{X:2});value:=Frozen{X:3};r:=new(value);_=p;_=q;_=r}`,
		`type Cell struct{im.Frozen;P *Frozen;Value Frozen};func f()*Frozen{return Cell{P:&Frozen{}}.P}`,
		`func f(){p:=&Frozen{};values:=[1]*Frozen{p};_=values[0];builder:=[]*Frozen{p};_=im.FreezeSlice(builder)}`,
		`func f(){p:=&Frozen{};values:=map[*Frozen]*Frozen{p:p};frozen:=im.FreezeMap(values);v,ok:=frozen.Get(p);_=v;_=ok}`,
		`func f(){var p *Frozen;p=nil;_=readSnapshot(nil);_=readSnapshot(p)}`,
		`func f()Local {p:=&Local{X:1};q:=p;p=&Local{X:2};_=readLocal(p);return *q}`,
		`func f(){var p *Local;p=new(Local);q:=new(Local{X:1});v:=Local{X:2};r:=new(v);_=readLocal(p);_=q;_=r}`,
		`func f(){var p *Local;p=nil;_=readLocal(nil);_=readLocal(p)}`,
		`type Alias=Local;func f(){p:=&Alias{};q:=p;_=readLocal(q)}`,
		`func f(){values:=[]Frozen{{X:1}};values[0]=Frozen{X:2};_=im.FreezeSlice(values)}`,
		`type Box[T any]struct{V T};func f(){_=Box[int]{V:1}}`,
		`func copyScalar[T any](v T)T{return v};func f(v Frozen){_=copyScalar(v.X)}`,
		`func f[T any](){v:=Frozen{};_=v}`,
		`func f[T any](){v:=Local{};_=v}`,
		`type Box[T any]struct{V T};func(b Box[T])Read()int{v:=Frozen{};return v.X}`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, immutableTestTypes+source); got != "" {
				t.Fatalf("supported immutable form rejected: %s", got)
			}
		})
	}
}

// TestImmutableObjectBoundaries checks writes, reference escape, and unsupported type shapes.
func TestImmutableObjectBoundaries(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"field", `func f(){v:=Frozen{};v.X=1}`, "writes to immutable objects"},
		{"array field", `func f(){v:=Frozen{};v.Array[0]=1}`, "writes to immutable objects"},
		{"pointer field", `func f(){p:=&Frozen{};p.X++}`, "writes to immutable objects"},
		{"pointer object", `func f(){p:=&Frozen{};*p=Frozen{X:1}}`, "writes to immutable objects"},
		{"Local field", `func f(){p:=&Local{};p.X=1}`, "writes to immutable objects"},
		{"builder object field", `func f(){v:=[]Frozen{{}};v[0].X=1;_=im.FreezeSlice(v)}`, "writes to immutable objects"},
		{"named address", `func f(){v:=Frozen{};p:=&v;_=p}`, "fresh allocation"},
		{"projection pointer", `func f(){p:=new(im.SliceProjection[int]);_=p}`, "concrete marked struct target"},
		{"array pointer", `func f(){p:=new([2]Frozen);_=p}`, "concrete marked struct target"},
		{"Local return", `func f()*Local{return &Local{}}`, "returning Local pointers"},
		{"Local nil result", `func f()*Local{return nil}`, "returning Local pointers"},
		{"Local global", `var p *Local`, "package variables"},
		{"Local stored field", `type Cell struct{im.Frozen;P *Local}`, "Frozen storage cannot contain Local values"},
		{"Local stored array", `func f(){p:=&Local{};_=[1]*Local{p}}`, "Local"},
		{"Local stored slice", `func f(){p:=&Local{};values:=[]*Local{p};_=len(values)}`,
			"containers of Local pointers"},
		{"Local capture", `func f(){p:=&Local{};_=func()int{return p.X}}`, "closures require"},
		{"shared capture", `func f(){p:=&Frozen{};_=func()int{return p.X}}`, ""},
		{"shared asynchronous", `func f(){p:=&Frozen{};go readSnapshot(p)}`, ""},
		{"mixed markers", `type Mixed struct{im.Frozen;im.Local}`, "cannot embed both"},
		{"generic marked", `type Box[T any]struct{im.Frozen;X int}`, "user generic declarations"},
		{"generic body write", `func f[T any](){v:=Frozen{};v.X++}`, "writes to immutable objects"},
		{
			"generic collection",
			`func f[T ~int](v T)im.FrozenSlice[T]{s:=im.FreezeSlice([]T{v});return s}`,
			"type parameters are unsupported in immutable storage",
		},
		{"generic instance", `type Box[T any]struct{V T};var value Box[Frozen]`, "unmarked struct"},
		{
			"generic body Local capture",
			`func f[T any]()func()int{v:=Local{};return func()int{return v.X}}`,
			"closures require",
		},
		{"Local nil builder", `func f(){values:=[]*Local{nil};_=len(values)}`, "Local"},
		{"Local empty builder", `func f(){var values []*Local;_=len(values)}`, "Local"},
		{
			"range interface",
			`func f(){values:=im.FreezeMap(map[Frozen]int{{}:1});var boxed any;for boxed=range values.All(){};_=boxed}`,
			"",
		},
		{"reference comparison box", `func f(p *Frozen,boxed any)bool{return p==boxed}`, ""},
		{"value comparison box", `func f(v Frozen,boxed any)bool{return boxed!=v}`, ""},
		{
			"tuple callback",
			`func pair()(Frozen,int){return Frozen{},1};func f(callback func(Frozen,int)){callback(pair())}`,
			"",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got := checkSource(t, immutableTestTypes+test.source)
			if (test.want == "" && got != "") || !strings.Contains(got, test.want) {
				t.Fatalf("diagnostics = %s; want %s", got, test.want)
			}
		})
	}
}
