package analyzer

import (
	"strings"
	"testing"
)

// TestSubsetBoundaries checks the transports excluded by the immutable reference contract.
func TestSubsetBoundaries(t *testing.T) {
	cases := []struct{ name, source string }{
		{"assertion", `func f(x any){_=x.(*V)}`},
		{"pointer conversion", `func f(p *V){_=(*V)(p)}`},
		{"tuple box", `func pair()(V,int){return V{},0};func f(){var x any;var n int;x,n=pair();_,_=x,n}`},
		{"tuple call box", `func pair()(V,int){return V{},0};func sink(any,int){};func f(){sink(pair())}`},
		{"shared child reborrow", `type Parent struct{im.Shared;Child V};
 func f(){var owner Parent;p:=new(owner);q:=&p.Child;_=q.X}`},
		{"scalar address", `func f(){var v V; _=&v.X}`},
		{"native slice", `func f(){_=[]V{{}}}`},
		{"native map", `func f(){_=map[int]V{0:{}}}`},
		{"native channel", `func f(){_=make(chan V)}`},
		{"values output", `func f(){_=im.FreezeSlice([]V{{}}).Clone()}`},
		{"indirection", `func f(v **V){}`},
		{"closure", `func f(){var v V; _=func(){_=v.X}}`},
		{"box", `func f(){var v V; var x any=v; _=x}`},
		{"box return", `func f()any{return V{}}`},
		{"generic receiver", `type Box[T ~int]struct{im.Shared;X T};
 func(b *Box[T])Read()T{return b.X};func f(){var b Box[int];_=b.Read()}`},
		{"generic", `func read[T any](v *T){};func f(){var v V; read(new(v))}`},
		{"dynamic", `func f(read func(*V)){var v V;read(new(v))}`},
		{"defer", `func read(v *V){};func f(){var v V;defer read(new(v))}`},
		{"goroutine", `func read(v *V){};func f(){var v V;go read(new(v))}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := checkSource(t, `type V struct{im.Shared;X int};`+tc.source)
			if !strings.Contains(got, "\n  source: ") || !strings.Contains(got, "\n  fix: ") {
				t.Fatalf("missing actionable diagnostic context: %s", got)
			}
			if !strings.Contains(got, "unsupported immutable operation") {
				t.Fatalf("missing subset rejection: %s", got)
			}
		})
	}
}

// TestSubsetValues checks copies and sealed constructors with marked values.
func TestSubsetValues(t *testing.T) {
	cases := []string{
		`func f(){v:=V{X:1};copy:=v;copy=V{X:2};_=copy}`,
		`func read(v *V)int{return v.X};func f(){var v V;_=read(new(v));v=V{X:2}}`,
		`func f(){_=im.FreezeSlice([]V{{},{}})}`,
		`func f(){_=im.FreezeSlice([]V{{}})}`,
		`func f(){_=im.FreezeMap(map[int]V{0:{}})}`,
		`func f(){values:=[2]V{{X:2},{}};_=values}`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, `type V struct{im.Shared;X int};`+source); got != "" {
				t.Fatalf("unexpected diagnostics: %s", got)
			}
		})
	}
}

// TestReferenceOperationAllowlist covers each documented reference producer and use.
func TestReferenceOperationAllowlist(t *testing.T) {
	cases := []string{
		`func f(){var v V;p:=new(v);_=p.X}`,
		`type Parent struct{im.Shared;Child V};func f(){var v Parent;p:=new(v.Child);_=p.X}`,
		`func f(){var v V;var p *V=new(v);_=p.X}`,
		`func f(p *V)V{return *p}`,
		`func f(p,q *V){_=p==q;_=p!=nil;_=nil==p}`,
		`func read(p *V)int{return p.X};func f(){_=read((&V{}))}`,
		`func Read(p *V)int{return p.X};func f(){var v V;p:=new(v);_=Read(p)}`,
		`type Alias=V;func read(p *Alias)int{return p.X};func f(){var v V;p:=new(v);_=read(p)}`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, `type V struct{im.Shared;X int};`+source); got != "" {
				t.Fatalf("allowlisted operation failed: %s", got)
			}
		})
	}
}

// TestReferenceOperationDefaultDenial covers valid Go operations absent from the supported table.
func TestReferenceOperationDefaultDenial(t *testing.T) {
	cases := []string{
		`func f(){p:=new([2]V);_=len(p)}`,
		`func f(){p:=new([2]V);_=p[0]}`,
		`func f(){p:=new([2]V);for range p{}}`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			got := checkSource(t, `type V struct{im.Shared;X int};`+source)
			if !strings.Contains(got, "immutable references require a concrete marked struct target") {
				t.Fatalf("unlisted reference operation was accepted: %s", got)
			}
		})
	}
}
