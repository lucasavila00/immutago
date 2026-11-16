package analyzer

import (
	"strings"
	"testing"
)

const frozenPointerStorageTypes = `
type Count int
func (c *Count) Inc(){*c++}
type Score struct { im.Copied; N int }
func(s *Score) Inc(){s.N++}
type Data struct { im.Frozen; N int; Count Count; Score Score; Array [2]int }
func (d *Data) Read()int{return d.N}
func (d Data) Value()int{return d.N}
type Alias = Data
type Pointer = *Alias
type State struct { Current Pointer; Next *State; Calls int }
func (s *State) Set(p *Data){s.Current=p;s.Calls++}
type Embedded struct { *Data; Calls int }
type Outer struct { Embedded }
var current im.Cell[Data]
`

// TestFrozenPointerStorage accepts ordinary carriers without builder ownership.
func TestFrozenPointerStorage(t *testing.T) {
	cases := []string{
		// Generic calls use ordinary API contracts; this check does not prove their mutation effects.
		`func f[T any](p *T,v T){*p=v};func g(p *Data){f(p,Data{})}`,
		`func f(){var s State;s.Current=current.LoadPointer().Unchecked();s.Current=nil;s.Set(&Data{});_=s.Current.N}`,
		`func f(p *Data){s:=State{Current:p};s.Calls++;copy:=s;copy.Current=nil;_=s.Current.Read()}`,
		`func f(p *Data)any{s:=&State{Current:p};var boxed any=s;return boxed}`,
		`func f(p *Data){s:=State{Current:p};fn:=func(){s.Calls++;_=s.Current.N};go fn();defer fn()}`,
		`func f(p *Data){a:=[]*Data{p};b:=a;b[0]=nil;a=append(a,p);_=a[1].N;clear(b)}`,
		`func f(p *Data){a:=map[string]*Data{"x":p};b:=a;b["x"]=nil;a["y"]=p;_=a["y"].N;delete(b,"x")}`,
		`func f(p *Data){a:=map[*Data]int{p:1};b:=a;b[p]=2;_=a[p];delete(b,p)}`,
		`func f(p *Data){a:=map[string]*Data{"x":p};q,ok:=a["x"];q,ok=a["y"];var r,found=a["x"];_=q;_=ok;_=r;_=found}`,
		`func f(p *Data){a:=map[*Data]int{p:1};n,ok:=a[p];_=n;_=ok}`,
		`func f(p *Data){a:=[2]*Data{p,nil};b:=&a;b[0]=nil;_=a[1]}`,
		`func f(p *Data){var s Outer;s.Data=p;s.Calls++;_=s.N;s.Data=nil}`,
		`func f(p *Data){var s struct{States []State};s.States=[]State{{Current:p}};s.States[0].Current=nil}`,
		`var saved *Data;func f(){saved=current.LoadPointer().Unchecked()}`,
		`func f()[]*Data{return []*Data{current.LoadPointer().Unchecked()}}`,
		`func f(s *State)any{return s.Current}`,
		`func id[T any](v T)T{return v};func f(s *State){_=id(s.Current)}`,
		`func f(s *State){slot:=&s.Current;*slot=nil;*slot=current.LoadPointer().Unchecked()}`,
		`func f(s *State){_=s.Current.Value();read:=s.Current.Read;_=read()}`,
		`func f(p *Data){_=(*Data).Read(p);read:=(*Data).Read;_=read(p)}`,
		`func f(s *Outer){_=s.Read()}`,
		`func f(p *Data){var box any=p;q:=box.(*Data);_=q.N}`,
		`func f(p *Data){read:=func()int{return p.N};go read();defer read()}`,
		`func f(p *Data){ch:=make(chan *Data,1);ch<-p;q,ok:=<-ch;_=ok;_=q.N}`,
		`func f(p *Data){var box any=p;switch v:=box.(type){case *Data:_=v.N}}`,
		`type P *Data;func f(p P){q:=(*Data)(p);_=q.N;_=p.N}`,
		`func f(s *State){score:=s.Current.Score;score.Inc()}`,
		`func f(){a:=[]*Data{{N:1}};_=a[0].N}`,
		`func get[T ~*Data](p T)int{return (*p).N};func f(p *Data)int{return get(p)}`,
		`type Bound interface{~*Alias};type Box[T Bound]struct{Current T}
func get[T Bound](p T)T{var zero T;p=zero;return p};func f(p Pointer){_=get(p);_=Box[Pointer]{Current:p}}`,
		`func get[T ~*Data](p T)T{p=nil;return p};func f(p Pointer){_=get(p)}`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, frozenPointerStorageTypes+source); got != "" {
				t.Fatalf("ordinary pointer storage rejected:\n%s", got)
			}
		})
	}
}

// TestFrozenPointerTargets checks explicit and promoted target operations through carriers.
func TestFrozenPointerTargets(t *testing.T) {
	cases := []struct{ source, want string }{
		{`func f(s *State){s.Current.N=1}`, "writes to immutable objects"},
		{`func f(s *State){*s.Current=Data{}}`, "writes to immutable objects"},
		{`func f(s *State){s.Current.Array[0]++}`, "writes to immutable objects"},
		{`func f(a []*Data){a[0].N++}`, "writes to immutable objects"},
		{`func f(a map[string]*Data){a["x"].N++}`, "writes to immutable objects"},
		{`func f(s *Outer){s.N++}`, "writes to immutable objects"},
		{`func f(s *Outer){s.Array[0]=1}`, "writes to immutable objects"},
		{`func f(s *State){_= &s.Current.N}`, "interior addresses"},
		{`func f(s *Outer){_= &s.N}`, "interior addresses"},
		{`func f(s *State){s.Current.Count.Inc()}`, "receivers must match"},
		{`func f(s *Outer){s.Count.Inc()}`, "receivers must match"},
		{`func f(s *State){s.Current.Score.Inc()}`, "receivers must match"},
		{`func f(s *Outer){s.Score.Inc()}`, "receivers must match"},
		{`func f(p *Data){var box any=p;q:=box.(*Data);q.N++}`, "writes to immutable objects"},
		{`type P *Data;func f(p P){p.N++}`, "writes to immutable objects"},
		{`type P *Data;func f(p P){_= &p.N}`, "interior addresses"},
		{`func f(s *State){slot:=&s.Current;(**slot).N++}`, "writes to immutable objects"},
		{`func f[T ~*Data](p T){(*p).N++}`, "writes to immutable objects"},
		{`func f[T ~*Data](p T){*p=Data{}}`, "writes to immutable objects"},
		{`func f[T ~*Data](p T){(*p).Array[0]++}`, "writes to immutable objects"},
		{`func f[T ~*Data](p T){_= &(*p).N}`, "interior addresses"},
		{`func f[T ~*Data](p T){(*p).Count.Inc()}`, "receivers must match"},
		{`func f(v Data){_= &v}`, "fresh allocation"},
		{`type Bad struct { Value Data }`, "unmarked struct field"},
		{`type Result struct{im.Local;N int};type Bad struct { Value *Result }`, "unmarked struct"},
		{`func f(){var s struct{ Value im.Projection[*Data] };_=s}`, "unmarked struct"},
		{`func f(){a:=[]*Data{&Data{}};b:=a;_=im.FreezeSlice(a);_=b}`, "builder"},
	}
	for _, test := range cases {
		t.Run(test.source, func(t *testing.T) {
			if got := checkSource(t, frozenPointerStorageTypes+test.source); !strings.Contains(got, test.want) {
				t.Fatalf("diagnostics = %q; want %q", got, test.want)
			}
		})
	}
}

// TestCellPointerPayload requires a concrete Frozen record for Frozen pointer loads.
func TestCellPointerPayload(t *testing.T) {
	for _, payload := range []string{"int", "*Data", "[2]Data", "im.FrozenString"} {
		source := "var raw im.Cell[" + payload + "];func f(){_=raw.LoadPointer().Unchecked()}"
		if got := checkSource(t, frozenPointerStorageTypes+source); !strings.Contains(got, "concrete Frozen record target") {
			t.Fatalf("payload %s: diagnostics = %q", payload, got)
		}
	}
}
