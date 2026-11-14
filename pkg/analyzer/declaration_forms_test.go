package analyzer

import (
	"strings"
	"testing"
)

// TestDeclaredStorageForms checks the contract when no protected runtime source exists.
func TestDeclaredStorageForms(t *testing.T) {
	cases := []struct{ source, want string }{
		{`var p *im.FrozenSlice[int]`, "concrete marked struct target"},
		{`func read(*[1]Value){};func f(){read(nil)}`, "concrete marked struct target"},
		{`func f()(p *[1]Value){return}`, "concrete marked struct target"},
		{`var callback func(Value)`, "immutable callbacks"},
		{`func read(callback func(Value)){};func f(){read(nil)}`, "immutable callbacks"},
		{`func f()(callback func(Value)){return}`, "immutable callbacks"},
		{`type Holder struct{Callback func(Value)};var holder Holder`, "immutable callbacks"},
		{`var callbacks chan func(Value)`, "immutable callbacks"},
		{`var callback *func(Value)`, "immutable callbacks"},
		{`type Holder struct{Callback func(Value)};func f()(holder Holder){return}`, "immutable callbacks"},
		{`type Callback=func(Value);type Holder struct{Next *Holder;Items [1]Callback};var holder Holder`,
			"immutable callbacks"},
		{`type Callback=func(Value);var values map[CallbackKey][]Callback;type CallbackKey int`, "immutable callbacks"},
		{`var callback interface{Read(Value)}`, "immutable callbacks"},
		{`type Other Value;func f(value Value)Other{return Other(value)}`, "protected value conversions"},
		{`type Other im.FrozenSlice[int];func f(value im.FrozenSlice[int])Other{return Other(value)}`,
			"protected value conversions"},
	}
	for _, test := range cases {
		t.Run(test.source, func(t *testing.T) {
			got := checkSource(t, `type Value struct{im.Shared;X int};`+test.source)
			if !strings.Contains(got, test.want) {
				t.Fatalf("unsupported declaration or conversion accepted: %s", got)
			}
		})
	}
}

// TestConstantMetadataHasNoRuntimeStorage uses the compiler's constant proof, not a type-subtree exemption.
func TestConstantMetadataHasNoRuntimeStorage(t *testing.T) {
	cases := []string{
		`type Holder struct{im.Shared;Values [len([1]Value{})]int}`,
		`var first,second [len([1]Value{})]int`,
		`func f(){first,second:=len([1]Value{}),2;_,_=first,second}`,
		`func f[T any](){var values [len([1]Value{})]int;_=values}`,
	}
	for _, source := range cases {
		if got := checkSource(t, `type Value struct{im.Shared;X int};`+source); got != "" {
			t.Fatalf("constant metadata rejected: %s", got)
		}
	}
	got := checkSource(t, `type Value struct{im.Shared;X int};func makeValue()Value{return Value{}};
 func f(){first,second:=len([1]Value{makeValue()}),2;_,_=first,second}`)
	if !strings.Contains(got, "multiple assignments involving protected storage") {
		t.Fatalf("evaluated protected construction skipped: %s", got)
	}
}

// TestTrackedBuilderDefinitionsMatchWholeBindings covers every local initialization form.
func TestTrackedBuilderDefinitionsMatchWholeBindings(t *testing.T) {
	for _, source := range []string{
		`func f(){values,n:=[]int{1},2;_=n;_=im.FreezeSlice(values)}`,
		`func f(){values,other:=[]int{1},[]int{2};_=im.FreezeSlice(values);_=im.FreezeSlice(other)}`,
		`func f(){values,n:=make([]int,1),2;_=n;_=im.FreezeSlice(values)}`,
	} {
		if got := checkSource(t, source); !strings.Contains(got, "multiple assignments involving protected storage") {
			t.Fatalf("multiple builder bindings accepted: %s", got)
		}
	}
}
