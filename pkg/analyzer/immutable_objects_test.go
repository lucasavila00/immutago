package analyzer

import (
	"strings"
	"testing"
)

const immutableTestTypes = `
type Snapshot struct { im.Snapshot; X int; Array [2]int }
type Local struct { im.Local; X int }
func readSnapshot(value *Snapshot) int { if value == nil { return 0 }; return value.X }
func readLocal(value *Local) int { if value == nil { return 0 }; return value.X }
`

// TestImmutableObjectForms checks fresh construction, binding replacement, and the two reference modes.
func TestImmutableObjectForms(t *testing.T) {
	cases := []string{
		`type Reader int;func(Reader)Read(p *Snapshot)int{return p.X};func f(){p:=&Snapshot{};_=Reader(0).Read(p)}`,
		`type Reader int;func(Reader)Make()*Snapshot{return &Snapshot{}};func f(){_=Reader(0).Make()}`,

		`func f()*Snapshot { p:=&Snapshot{X:1}; q:=p; p=&Snapshot{X:2}; _=p; return q }`,
		`var global *Snapshot;func f(){global=&Snapshot{X:1};global=nil}`,
		`var global Snapshot;func f(){global=Snapshot{X:1}}`,
		`func f(){p:=new(Snapshot);q:=new(Snapshot{X:2});value:=Snapshot{X:3};r:=new(value);_=p;_=q;_=r}`,
		`type Cell struct{im.Snapshot;P *Snapshot;Value Local};func f()*Snapshot{return Cell{P:&Snapshot{}}.P}`,
		`func f(){p:=&Snapshot{};values:=[1]*Snapshot{p};_=values[0];builder:=[]*Snapshot{p};_=im.FreezeSlice(builder)}`,
		`func f(){p:=&Snapshot{};values:=map[*Snapshot]*Snapshot{p:p};frozen:=im.FreezeMap(values);v,ok:=frozen.Get(p);_=v;_=ok}`,
		`func f(){var p *Snapshot;p=nil;_=readSnapshot(nil);_=readSnapshot(p)}`,
		`func f()Local {p:=&Local{X:1};q:=p;p=&Local{X:2};_=readLocal(p);return *q}`,
		`func f(){var p *Local;p=new(Local);q:=new(Local{X:1});v:=Local{X:2};r:=new(v);_=readLocal(p);_=q;_=r}`,
		`func f(){var p *Local;p=nil;_=readLocal(nil);_=readLocal(p)}`,
		`type Alias=Local;func f(){p:=&Alias{};q:=p;_=readLocal(q)}`,
		`func f(){values:=[]Snapshot{{X:1}};values[0]=Snapshot{X:2};_=im.FreezeSlice(values)}`,
		`type Box[T any]struct{V T};func f(){_=Box[int]{V:1}}`,
		`func copyScalar[T any](v T)T{return v};func f(v Snapshot){_=copyScalar(v.X)}`,
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
		{"field", `func f(){v:=Snapshot{};v.X=1}`, "writes to immutable objects"},
		{"array field", `func f(){v:=Snapshot{};v.Array[0]=1}`, "writes to immutable objects"},
		{"pointer field", `func f(){p:=&Snapshot{};p.X++}`, "writes to immutable objects"},
		{"pointer object", `func f(){p:=&Snapshot{};*p=Snapshot{X:1}}`, "writes to immutable objects"},
		{"Local field", `func f(){p:=&Local{};p.X=1}`, "writes to immutable objects"},
		{"builder object field", `func f(){v:=[]Snapshot{{}};v[0].X=1;_=im.FreezeSlice(v)}`, "writes to immutable objects"},
		{"named address", `func f(){v:=Snapshot{};p:=&v;_=p}`, "fresh allocation"},
		{"frozen pointer", `func f(){p:=new(im.FrozenSlice[int]);_=p}`, "concrete marked struct target"},
		{"array pointer", `func f(){p:=new([2]Snapshot);_=p}`, "concrete marked struct target"},
		{"Local return", `func f()*Local{return &Local{}}`, "returning Local pointers"},
		{"Local nil result", `func f()*Local{return nil}`, "returning Local pointers"},
		{"Local global", `var p *Local`, "global Local pointers"},
		{"Local stored field", `type Cell struct{im.Snapshot;P *Local}`, "Local pointers cannot be stored"},
		{"Local stored array", `func f(){p:=&Local{};_=[1]*Local{p}}`, "Local"},
		{"Local stored slice", `func f(){p:=&Local{};values:=[]*Local{p};_=len(values)}`,
			"containers of Local pointers"},
		{"Local capture", `func f(){p:=&Local{};_=func()int{return p.X}}`, "closures cannot capture"},
		{"shared capture", `func f(){p:=&Snapshot{};_=func()int{return p.X}}`, "closures cannot capture"},
		{"shared asynchronous", `func f(){p:=&Snapshot{};go readSnapshot(p)}`, "synchronous"},
		{"mixed markers", `type Mixed struct{im.Snapshot;im.Local}`, "cannot embed both"},
		{"generic marked", `type Box[T any]struct{im.Snapshot;X int}`, "user generic declarations"},
		{"generic local", `func f[T any](){v:=Snapshot{};_=v}`, "user generic declarations"},
		{
			"generic collection",
			`func f[T ~int](v T)im.FrozenSlice[T]{s:=im.FreezeSlice([]T{v});return s}`,
			"user generic declarations",
		},
		{"generic instance", `type Box[T any]struct{V T};var value Box[Snapshot]`, "user generic instantiations"},
		{
			"generic receiver body",
			`type Box[T any]struct{V T};func(b Box[T])Read()int{v:=Snapshot{};return v.X}`,
			"user generic declarations",
		},
		{"Local nil builder", `func f(){values:=[]*Local{nil};_=len(values)}`, "Local"},
		{"Local empty builder", `func f(){var values []*Local;_=len(values)}`, "Local"},
		{
			"range interface",
			`func f(){values:=im.FreezeMap(map[Snapshot]int{{}:1});var boxed any;for boxed=range values.All(){};_=boxed}`,
			"boxing immutable storage",
		},
		{"reference comparison box", `func f(p *Snapshot,boxed any)bool{return p==boxed}`, "boxing immutable storage"},
		{"value comparison box", `func f(v Snapshot,boxed any)bool{return boxed!=v}`, "boxing immutable storage"},
		{
			"tuple callback",
			`func pair()(Snapshot,int){return Snapshot{},1};func f(callback func(Snapshot,int)){callback(pair())}`,
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
