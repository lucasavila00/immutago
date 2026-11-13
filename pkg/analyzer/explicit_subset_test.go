package analyzer

import (
	"strings"
	"testing"
)

// TestExplicitSubsetRejects checks every excluded statement and reference form.
func TestExplicitSubsetRejects(t *testing.T) {
	cases := []struct{ name, source string }{
		{"method", `func(v V)Read()int{return v.X}`},
		{"promoted method", `type Count int;func(c Count)Read()int{return int(c)}
   type W struct{im.Immut;Count};func f(){var w W;_=w.Read()}`},
		{"range protected value", `func empty(yield func(V)bool){};func f(){for v:=range empty{_=v.X}}`},
		{"native multiple", `func f(){s:=[]int{1};var x int;s[0],x=2,3;_=x;_=im.FreezeSlice(s)}`},
		{"protected send", `func f(ch chan int){var v V;ch<-v.X}`},
		{"range reference", `func empty(yield func(*V)bool){};func f(){for p:=range empty{_=p.X}}`},
		{"multiple write", `func f(){var v V;v.X,v.X=1,2}`},
		{"multiple value", `func f(){a,b:=V{},V{};_,_=a,b}`},
		{"existing Get", `func f(){m:=im.FreezeMap(map[int]V{});var v V;v,ok:=m.Get(1);_=v;_=ok}`},
		{"nested Freeze return", `func f()im.FrozenSlice[int]{return im.FreezeSlice([]int{1})}`},
		{"nested Freeze call", `func use(im.FrozenSlice[int]){};func f(){use(im.FreezeSlice([]int{1}))}`},
		{"nested Freeze chain", `func f(){_=im.FreezeSlice([]int{1}).Len()}`},
		{"select owner", `func f(ch chan bool){var v V;select{case <-ch:case ch<-(v.X==0):}}`},
		{"select builder", `func f(ch chan bool){s:=[]bool{true};select{case _,s[0]=<-ch:};_=im.FreezeSlice(s)}`},
		{"go value", `func use(V){};func f(){go use(V{})}`},
		{"defer value", `func use(V){};func f(){defer use(V{})}`},
		{"type switch", `func f(x any){switch v:=x.(type){case *V:_=v.X}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := checkSource(t, `type V struct{im.Immut;X int};`+tc.source)
			if !strings.Contains(got, "unsupported immutable operation") {
				t.Fatalf("unsupported form accepted: %s", got)
			}
		})
	}
}

// TestExplicitSubsetAccepts checks reference bindings, fresh map results, and explicit Freeze statements.
func TestExplicitSubsetAccepts(t *testing.T) {
	cases := []string{
		`func f(){p:=&V{};q:=p;p=&V{X:2};_=q;_=p}`,
		`func f(){var p *V=nil;_=p}`,
		`func f(){var p *V;_=p}`,
		`func f(p *V){p=nil}`,
		`func read(p *V)int{return p.X};func f(){var v V;var p *V=new(v);_=read(p);v=V{X:2}}`,
		`func f(){m:=im.FreezeMap(map[int]V{});v,ok:=m.Get(1);_=v;_=ok}`,
		`func f(){m:=im.FreezeMap(map[int]V{});var v,ok=m.Get(1);_=v;_=ok}`,
		`var frozen=im.FreezeSlice([]int{1})`,
		`func f(){_=im.FreezeSlice([]int{1})}`,
		`func f(ch chan bool){var v V;select{case <-ch:v=V{X:2}};_=v}`,
		`func f(){var v V;switch v.X{case 0:v=V{X:2}};_=v}`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, `type V struct{im.Immut;X int};`+source); got != "" {
				t.Fatalf("documented form rejected: %s", got)
			}
		})
	}
}
