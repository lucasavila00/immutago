package analyzer

import (
	"strings"
	"testing"
)

const immutableTestTypes = `
type Shared struct { im.Shared; X int; Array [2]int }
type Local struct { im.LocalRef; X int }
func readShared(value *Shared) int { if value == nil { return 0 }; return value.X }
func readLocal(value *Local) int { if value == nil { return 0 }; return value.X }
`

// TestImmutableObjectForms checks fresh construction, binding replacement, and the two reference modes.
func TestImmutableObjectForms(t *testing.T) {
	cases := []string{
		`func f()*Shared { p:=&Shared{X:1}; q:=p; p=&Shared{X:2}; _=p; return q }`,
		`var global *Shared;func f(){global=&Shared{X:1};global=nil}`,
		`var global Shared;func f(){global=Shared{X:1}}`,
		`func f(){p:=new(Shared);q:=new(Shared{X:2});value:=Shared{X:3};r:=new(value);_=p;_=q;_=r}`,
		`type Owner struct{im.Shared;P *Shared;Value Local};func f()*Shared{return Owner{P:&Shared{}}.P}`,
		`func f(){p:=&Shared{};values:=[1]*Shared{p};_=values[0];builder:=[]*Shared{p};_=im.FreezeSlice(builder)}`,
		`func f(){p:=&Shared{};values:=map[*Shared]*Shared{p:p};frozen:=im.FreezeMap(values);v,ok:=frozen.Get(p);_=v;_=ok}`,
		`func f(){var p *Shared;p=nil;_=readShared(nil);_=readShared(p)}`,
		`func f()Local {p:=&Local{X:1};q:=p;p=&Local{X:2};_=readLocal(p);return *q}`,
		`func f(){var p *Local;p=new(Local);q:=new(Local{X:1});v:=Local{X:2};r:=new(v);_=readLocal(p);_=q;_=r}`,
		`func f(){var p *Local;p=nil;_=readLocal(nil);_=readLocal(p)}`,
		`type Alias=Local;func f(){p:=&Alias{};q:=p;_=readLocal(q)}`,
		`func f(){values:=[]Shared{{X:1}};values[0]=Shared{X:2};_=im.FreezeSlice(values)}`,
		`type Box[T any]struct{V T};func f(){_=Box[int]{V:1}}`,
		`func copyScalar[T any](v T)T{return v};func f(v Shared){_=copyScalar(v.X)}`,
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
		{"field", `func f(){v:=Shared{};v.X=1}`, "writes to immutable objects"},
		{"array field", `func f(){v:=Shared{};v.Array[0]=1}`, "writes to immutable objects"},
		{"pointer field", `func f(){p:=&Shared{};p.X++}`, "writes to immutable objects"},
		{"pointer object", `func f(){p:=&Shared{};*p=Shared{X:1}}`, "writes to immutable objects"},
		{"NoRef field", `func f(){p:=&Local{};p.X=1}`, "writes to immutable objects"},
		{"builder object field", `func f(){v:=[]Shared{{}};v[0].X=1;_=im.FreezeSlice(v)}`, "writes to immutable objects"},
		{"named address", `func f(){v:=Shared{};p:=&v;_=p}`, "fresh allocation"},
		{"frozen pointer", `func f(){p:=new(im.FrozenSlice[int]);_=p}`, "concrete marked struct target"},
		{"array pointer", `func f(){p:=new([2]Shared);_=p}`, "concrete marked struct target"},
		{"NoRef return", `func f()*Local{return &Local{}}`, "returning LocalRef references"},
		{"NoRef nil result", `func f()*Local{return nil}`, "returning LocalRef references"},
		{"NoRef global", `var p *Local`, "global LocalRef references"},
		{"NoRef stored field", `type Owner struct{im.Shared;P *Local}`, "LocalRef references cannot be stored"},
		{"NoRef stored array", `func f(){p:=&Local{};_=[1]*Local{p}}`, "LocalRef"},
		{"NoRef stored slice", `func f(){p:=&Local{};values:=[]*Local{p};_=len(values)}`, "reference operation"},
		{"NoRef capture", `func f(){p:=&Local{};_=func()int{return p.X}}`, "closures cannot capture"},
		{"shared capture", `func f(){p:=&Shared{};_=func()int{return p.X}}`, "closures cannot capture"},
		{"shared asynchronous", `func f(){p:=&Shared{};go readShared(p)}`, "synchronous"},
		{"mixed markers", `type Mixed struct{im.Shared;im.LocalRef}`, "cannot embed both"},
		{"generic marked", `type Box[T any]struct{im.Shared;X int}`, "user generic declarations"},
		{"generic local", `func f[T any](){v:=Shared{};_=v}`, "user generic declarations"},
		{
			"generic collection",
			`func f[T ~int](v T)im.FrozenSlice[T]{s:=im.FreezeSlice([]T{v});return s}`,
			"user generic declarations",
		},
		{"generic instance", `type Box[T any]struct{V T};var value Box[Shared]`, "user generic instantiations"},
		{
			"generic receiver body",
			`type Box[T any]struct{V T};func(b Box[T])Read()int{v:=Shared{};return v.X}`,
			"user generic declarations",
		},
		{"NoRef nil builder", `func f(){values:=[]*Local{nil};_=len(values)}`, "LocalRef"},
		{"NoRef empty builder", `func f(){var values []*Local;_=len(values)}`, "LocalRef"},
		{
			"range interface",
			`func f(){values:=im.FreezeSlice([]Shared{{}});var boxed any;for _,boxed=range values.All(){};_=boxed}`,
			"boxing immutable storage",
		},
		{
			"method parameter",
			`type Reader int;func(Reader)Read(p *Shared)int{return p.X};func f(){p:=&Shared{};_=Reader(0).Read(p)}`,
			"user-defined methods with immutable storage",
		},
		{
			"method result",
			`type Reader int;func(Reader)Make()*Shared{return &Shared{}};func f(){_=Reader(0).Make()}`,
			"user-defined methods with immutable storage",
		},
		{"reference comparison box", `func f(p *Shared,boxed any)bool{return p==boxed}`, "boxing immutable storage"},
		{"value comparison box", `func f(v Shared,boxed any)bool{return boxed!=v}`, "boxing immutable storage"},
		{
			"tuple callback",
			`func pair()(Shared,int){return Shared{},1};func f(callback func(Shared,int)){callback(pair())}`,
			"dynamic calls with immutable storage",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := checkSource(t, immutableTestTypes+test.source); !strings.Contains(got, test.want) {
				t.Fatalf("diagnostics = %s; want %s", got, test.want)
			}
		})
	}
}
