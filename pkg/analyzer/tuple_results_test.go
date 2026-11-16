package analyzer

import (
	"strings"
	"testing"
)

const tupleResultTypes = `
type Local struct { im.Local; N int }
type Frozen struct { im.Frozen; N int }
type Count struct { im.Copied; N int }
type Alias = Local
func pair()(Local,bool){return Local{N:1},true}
func reverse()(bool,Local){return true,Local{N:1}}
func triple()(Local,Count,bool){return Local{N:1},Count{N:2},true}
`

// TestTupleResultForms checks each destination and forwarded result by its own type.
func TestTupleResultForms(t *testing.T) {
	cases := []struct{ name, source string }{
		{"local declarations", `func f(){value,ok:=pair();var other,found=pair();_=value;_=other;_,_=ok,found}`},
		{"named results", `func f()(value Alias,ok bool){value,ok=pair();return}`},
		{"zero named results", `func f()(value Alias,ok bool){return}`},
		{"return forwarding", `func f()(Alias,bool){return (pair)()}`},
		{"argument forwarding", `func take(Alias,bool){};func f(){take(pair())}`},
		{"variadic forwarding", `func values()(Local,bool,bool){return Local{},true,false};
func take(Local,...bool){};func f(){take(values())}`},
		{"ordinary field", `func f(){var value Local;var state struct{OK bool};value,state.OK=pair();_=value}`},
		{"ordinary global", `var found bool;func f(){var value Local;value,found=pair();_=value}`},
		{"ordinary first result", `var found bool;func f(){var value Local;found,value=reverse();_=value}`},
		{"ordinary pointer", `func f(out *bool){var value Local;value,*out=pair();_=value}`},
		{"ordinary element", `func f(out map[int]bool){var value Local;value,out[0]=pair();_=value}`},
		{"Copied field", `func f(){var value Local;var count Count;var ok bool;
value,count,ok=triple();_=value;count.N++;_=ok}`},
		{"package discard", `var _,found=pair()`},
		{"reused short declaration", `func f(){var ok bool;value,ok:=pair();_=value;_=ok}`},
		{"condition initializer", `func f(){if value,ok:=pair();ok{_=value}}`},
		{"ordinary callback", `func f(call func()(Count,bool))(Count,bool){value,ok:=call();return value,ok}`},
		{"Frozen callback", `func f(call func()(Frozen,bool))(Frozen,bool){return call()}`},
		{"ordinary Clone tuple", `func f(p im.SliceProjection[int])([]int,bool){return p.Clone(),true}`},
		{"scalar builder slot", `func f(){values:=[]bool{false};var value Local;
value,values[0]=pair();_=value;_=im.FreezeSlice(values)}`},
		{"protected builder slot", `func values()(Frozen,Local){return Frozen{N:2},Local{}};
func f(){items:=[]Frozen{{}};var value Local;items[0],value=values();_=value;_=im.FreezeSlice(items)}`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := checkSource(t, tupleResultTypes+test.source); got != "" {
				t.Fatalf("allowed tuple form rejected: %s", got)
			}
		})
	}
}

// TestTupleResultBoundaries keeps Local, write, call, and builder checks at each position.
func TestTupleResultBoundaries(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"Local package declaration", `var value,ok=pair()`, "package variables"},
		{"Local package assignment", `var value Local;func f(){var ok bool;value,ok=pair();_=ok}`, "package variables"},
		{"Local first boxed", `func f(){var value any;var ok bool;value,ok=pair();_,_=value,ok}`, "ordinary interfaces"},
		{"Local last boxed", `func f(){var value any;var ok bool;ok,value=reverse();_,_=value,ok}`, "ordinary interfaces"},
		{"Local argument boxed", `func take(bool,...any){};func f(){take(reverse())}`, "ordinary interfaces"},
		{"Local return boxed", `func f()(bool,any){return reverse()}`, "ordinary interfaces"},
		{"Local pointer named", `func f()(ok bool,value *Local){return}`, "returning Local pointers"},
		{"Local pointer array", `func f()(bool,[1]*Local){return false,[1]*Local{}}`, "containers of Local pointers"},
		{"Local callback", `func f(call func()(Local,bool)){_,_=call()}`, "dynamic calls"},
		{"Local captured result", `func f()func()int{value,_:=pair();return func()int{return value.N}}`, "captures"},
		{"protected first destination", `func f(){var value Local;var ok bool;value.N,ok=numbers();_=ok};
func numbers()(int,bool){return 1,true}`, "writes to immutable"},
		{"protected last destination", `func f(){var value Local;var ok bool;ok,value.N=numbers();_=ok};
func numbers()(bool,int){return true,1}`, "writes to immutable"},
		{"protected nested destination", `type Parent struct{im.Frozen;Count Count};
func numbers()(bool,int){return true,1};func f(p *Parent){var ok bool;ok,p.Count.N=numbers();_=ok}`,
			"writes to immutable"},
		{"consumed scalar slot", `func f(){values:=[]bool{false};_=im.FreezeSlice(values);
var value Local;value,values[0]=pair();_=value}`, "consumed builder"},
		{"consumed protected slot", `func values()(Frozen,Local){return Frozen{},Local{}};
func f(){items:=[]Frozen{{}};_=im.FreezeSlice(items);var value Local;items[0],value=values();_=value}`,
			"consumed builder"},
		{"independent expressions", `func f(){value,ok:=Local{},true;_=value;_=ok}`, "multiple assignments"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := checkSource(t, tupleResultTypes+test.source); !strings.Contains(got, test.want) {
				t.Fatalf("diagnostics = %s; want %s", got, test.want)
			}
		})
	}
}

// TestTupleBuilderBindings rejects tuple-produced handles at every binding position.
func TestTupleBuilderBindings(t *testing.T) {
	for _, element := range []string{"int", "Frozen"} {
		for _, form := range []string{"values,_:=pair()", "_,values:=reverse()", "var values,_=pair()",
			"var _,values=reverse()", "values:=[]E{};values,_=pair()", "values:=[]E{};_,values=reverse()"} {
			source := `type Frozen struct{im.Frozen;N int};
func pair()([]E,bool){return nil,true};func reverse()(bool,[]E){return true,nil};
func f(){` + form + `;_=im.FreezeSlice(values)}`
			source = strings.ReplaceAll(source, "[]E", "[]"+element)
			t.Run(element+"/"+form, func(t *testing.T) {
				if got := checkSource(t, source); !strings.Contains(got, "native builders require fresh local bindings") {
					t.Fatalf("tuple builder binding accepted: %s", got)
				}
			})
		}
	}
}
