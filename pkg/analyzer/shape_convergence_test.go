package analyzer

import (
	"strings"
	"testing"
)

// TestStatementShapeRejects checks complete bindings around nested protected operands.
func TestStatementShapeRejects(t *testing.T) {
	cases := []struct{ name, source string }{
		{"binary binding", `func f(){var v V;a,b:=v.X+1,2;_,_=a,b}`},
		{"binary declaration", `func f(){var v V;var a,b=v.X+1,2;_,_=a,b}`},
		{"call binding", `func f(){var v V;a,b:=read(new(v)),2;_,_=a,b}`},
		{"call declaration", `func f(){var v V;var a,b=read(new(v)),2;_,_=a,b}`},
		{"length binding", `func f(){s:=[]int{1};a,b:=len(s),2;_,_=a,b;_=im.FreezeSlice(s)}`},
		{"length declaration", `func f(){s:=[]int{1};var a,b=len(s),2;_,_=a,b;_=im.FreezeSlice(s)}`},
		{"nested builder binding", `func f(){s:=[]int{1};a,b:=(s[0]+1)*2,2;_,_=a,b;_=im.FreezeSlice(s)}`},
		{"nested builder declaration", `func f(){s:=[]int{1};var a,b=(s[0]+1)*2,2;_,_=a,b;_=im.FreezeSlice(s)}`},
		{"tuple result", `func pair()(V,int){return V{},0};func f(){v,n:=pair();_=v;_=n}`},
		{"range targets", `func f(){var v V;for v.X,v.Y=range []int{2}{};_=v}`},
		{"nested range index", `func f(){var v V;var a [1]int;var n int;for a[v.X],n=range []int{2}{};_=a;_=n}`},
		{"nested range call", `func f(){var v V;var a [1]int;var n int;for a[read(new(v))],n=range []int{2}{};_=a;_=n}`},
		{"nested range builder", `func f(){s:=[]int{0};var a [1]int;var n int;
   for a[s[0]],n=range []int{2}{};_=a;_=n;_=im.FreezeSlice(s)}`},
		{"range mixed targets", `func f(){var v V;var n int;for n,v.X=range []int{2}{};_=n}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := checkSource(t, `type V struct{im.Immut;X,Y int};func read(p *V)int{return p.X};`+tc.source)
			if !strings.Contains(got, "multiple assignments involving protected storage are unsupported") {
				t.Fatalf("multiple protected bindings accepted: %s", got)
			}
		})
	}
}

// TestStatementShapeAccepts checks single bindings, fresh Get results, and range value copies.
func TestStatementShapeAccepts(t *testing.T) {
	cases := []string{
		`func f(){var v V;a:=v.X+1;_=a}`,
		`func f(){var v V;var a=v.X+1;_=a}`,
		`func f(){s:=[]int{1};a:=len(s);_=a;_=im.FreezeSlice(s)}`,
		`func f(){m:=im.FreezeMap(map[int]V{});v,ok:=m.Get(0);_=v;_=ok}`,
		`func f(){m:=im.FreezeMap(map[int]V{});var v,ok=m.Get(0);_=v;_=ok}`,
		`func f(){s:=[]V{{X:1}};for i,v:=range s{_=i;_=v};_=im.FreezeSlice(s)}`,
		`func f(){var v V;var i int;for i=range []int{2}{v=V{X:i}};_=v}`,
		`func f(){a,b:=1+2,3;_,_=a,b}`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, `type V struct{im.Immut;X,Y int};`+source); got != "" {
				t.Fatalf("supported statement rejected: %s", got)
			}
		})
	}
}

// TestProtectedResultCalls rejects unknown producers of protected values and callbacks.
func TestProtectedResultCalls(t *testing.T) {
	cases := []struct{ name, source string }{
		{"value result", `func f(factory func()V)V{return factory()}`},
		{"frozen result", `func f(factory func()im.FrozenSlice[int])im.FrozenSlice[int]{return factory()}`},
		{"tuple result", `func f(factory func()(V,int))(V,int){return factory()}`},
		{"callback result", `func f(factory func()func()V){_=factory()}`},
		{"tuple callback result", `func f(factory func()(func()V,int)){factory()}`},
		{"interface result", `type Factory interface{Make()V};func f(factory Factory)V{return factory.Make()}`},
		{"field result", `type Factory struct{Make func()V};func f(factory Factory)V{return factory.Make()}`},
		{"literal result", `func f(){v:=func()V{return V{}}();_=v}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := checkSource(t, `type V struct{im.Immut;X int};`+tc.source)
			if !strings.Contains(got, "dynamic calls with immutable storage are unsupported") {
				t.Fatalf("unknown protected producer accepted: %s", got)
			}
		})
	}
	for _, source := range []string{
		`func f(callback func()int)int{return callback()}`,
		`func makeValue()V{return V{}};func f()V{return makeValue()}`,
		`type Factory interface{Make()int};func f(factory Factory)int{return factory.Make()}`,
	} {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, `type V struct{im.Immut;X int};`+source); got != "" {
				t.Fatalf("supported producer rejected: %s", got)
			}
		})
	}
}
