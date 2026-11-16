package analyzer

import (
	"strings"
	"testing"
)

// TestDirectCollectionCalls accepts parentheses and shared receiver reads.
func TestDirectCollectionCalls(t *testing.T) {
	cases := []string{
		`func read(p *V){};func f(){read(nil);read(((nil)))}`,
		`func read(p *V){};func f(){nil:=&V{};copy:=nil;read(copy)}`,
		`func f(){s:=im.FreezeSlice([]int{1});_=(s.Len)();_=((s.At))(0)}`,
		`func f(){s:=im.FreezeSlice([]int{1});p:=s;_=((p.Len))();_=(p.At)(0)}`,
		`func f(){m:=im.FreezeMap(map[int]V{});v,ok:=(m.Get)(1);_=v;_=ok}`,
		`func f(){m:=im.FreezeMap(map[int]V{});p:=m;v,ok:=((p.Get))(1);_=v;_=ok}`,
		`func read(p *V)bool{return p==(nil)};func f(){v:=V{};nil:=new(v);_=read(nil)}`,
		`func read(p *V)bool{return (nil)!=p};func f(){v:=V{};p:=new(v);q:=p;_=read((p));_=p==q}`,
		`func ordinary(p *int){};func f(){ordinary(nil)}`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, `type V struct{im.Frozen;X int};`+source); got != "" {
				t.Fatalf("allowed call rejected: %s", got)
			}
		})
	}
}

// TestDirectCollectionCallBoundaries checks nearby rejected forms and their rules.
func TestDirectCollectionCallBoundaries(t *testing.T) {
	cases := []struct{ source, diagnostic string }{
		{`func f(){frozen:=im.FreezeSlice([]int{1});
   s:=frozen.Projection();fn:=(s.Len);_=fn()}`, "method values require"},
		{
			`func f(){frozen:=im.FreezeSlice([]int{1});s:=frozen.Projection();_=im.SliceProjection[int].Len(s)}`,
			"method expressions require ordinary or persistent Frozen receivers and contracts",
		},
		{
			`func f(){frozen:=im.FreezeMap(map[int]V{});m:=frozen.Projection();v,ok:=im.MapProjection[int,V].Get(m,1);_=v;_=ok}`,
			"collection and projection calls require a supported receiver",
		},
		{
			`func f(){frozen:=im.FreezeMap(map[int]V{});m:=frozen.Projection();
   v,ok:=(im.MapProjection[int,V].Get)(m,1);_=v;_=ok}`,
			"collection and projection calls require a supported receiver",
		},
	}
	for _, test := range cases {
		t.Run(test.source, func(t *testing.T) {
			got := checkSource(t, `type V struct{im.Frozen;X int};`+test.source)
			if !strings.Contains(got, test.diagnostic) {
				t.Fatalf("missing %q: %s", test.diagnostic, got)
			}
		})
	}
}
