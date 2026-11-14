package analyzer

import (
	"strings"
	"testing"
)

// TestImplicitInterfaceConversions rejects boxing in variadic calls and switch cases.
func TestImplicitInterfaceConversions(t *testing.T) {
	cases := []struct{ name, source string }{
		{"variadic value", `func sink(...any){};func f(v V){sink(v)}`},
		{"variadic tail", `func sink(int,...any){};func f(v V){sink(1,v)}`},
		{"variadic callback", `func sink(...any){};func makeValue()V{return V{}};func f(){sink(makeValue)}`},
		{"tuple first", `func pair()(V,int){return V{},1};func sink(...any){};func f(){sink(pair())}`},
		{"tuple last", `func pair()(int,V){return 1,V{}};func sink(...any){};func f(){sink(pair())}`},
		{"tuple tail", `func pair()(int,V){return 1,V{}};func sink(int,...any){};func f(){sink(pair())}`},
		{"switch value", `func f(v V,boxed any){switch boxed{case v:}}`},
		{"switch expression", `func f(boxed any){switch boxed{case V{}:}}`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got := checkSource(t, `type V struct{im.Shared;X int};`+test.source)
			if !strings.Contains(got, "boxing immutable storage in interfaces is unsupported") &&
				!(test.name == "variadic callback" && strings.Contains(got, "immutable callbacks are unsupported")) {
				t.Fatalf("implicit interface conversion accepted: %s", got)
			}
		})
	}
}

// TestScalarInterfaceConversions preserves ordinary variadic calls and concrete value switches.
func TestScalarInterfaceConversions(t *testing.T) {
	cases := []string{
		`func sink(...any){};func f(v V){sink(v.X)}`,
		`func pair()(int,string){return 1,"x"};func sink(...any){};func f(){sink(pair())}`,
		`func pair()(V,int){return V{},1};func sink(V,int){};func f(){sink(pair())}`,
		`func sink(...any){};func f(){values:=[]any{1,"x"};sink(values...)}`,
		`func f(v V){switch v{case V{}:};switch v.X{case 1:}}`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, `type V struct{im.Shared;X int};`+source); got != "" {
				t.Fatalf("ordinary conversion rejected: %s", got)
			}
		})
	}
}
