package analyzer

import (
	"strings"
	"testing"
)

// TestClosureDeclarationScopes checks captures without using declaration order.
func TestClosureDeclarationScopes(t *testing.T) {
	cases := []string{
		`var read=func()int{return value.X};var value=&Value{X:1}`,
		`var value=&Value{X:1};var read=func()int{return value.X}`,
		`func f(){value:=&Value{};_=func()int{return value.X}}`,
		`func f(){_=func(){value:=&Value{};_=func()int{return value.X}}}`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			got := checkSource(t, `type Value struct{im.Shared;X int};`+source)
			if !strings.Contains(got, "closures cannot capture immutable storage") {
				t.Fatalf("capture accepted: %s", got)
			}
		})
	}
}

// TestClosureLocalConstruction distinguishes object fields from captured variables.
func TestClosureLocalConstruction(t *testing.T) {
	cases := []string{
		`var read=func()int{return count};var count=1`,
		`func f()int{return func()int{value:=&Value{X:1};return value.X}()}`,
		`type Parent struct{im.Shared;Child Value};func f()int{
 return func()int{value:=Parent{Child:Value{X:1}};return value.Child.X}()}`,
		`type Parent struct{im.Shared;Child *Value};func f()int{
 return func()int{value:=Parent{Child:&Value{X:1}};return value.Child.X}()}`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, `type Value struct{im.Shared;X int};`+source); got != "" {
				t.Fatalf("local construction or scalar global rejected: %s", got)
			}
		})
	}
}

// TestBuilderSourceActivatesWholeRange checks the full binding shape for scalar builder iteration.
func TestBuilderSourceActivatesWholeRange(t *testing.T) {
	got := checkSource(t, `func f(){values:=[]int{1};i:=0;v:=0;
 for i,v=range values{};_=i;_=v;_=im.FreezeSlice(values)}`)
	if !strings.Contains(got, "multiple assignments involving protected storage are unsupported") {
		t.Fatalf("two existing builder range targets accepted: %s", got)
	}
	for _, source := range []string{
		`func f(){values:=[]int{1};for i,v:=range values{_=i;_=v};_=im.FreezeSlice(values)}`,
		`func f(){values:=[]int{1};i:=0;for i=range values{};_=i;_=im.FreezeSlice(values)}`,
		`func f(){values:=[]int{1};i:=0;v:=0;for i,v=range values{};_=i;_=v};var _=im.Shared{}`,
	} {
		if got := checkSource(t, source); got != "" {
			t.Fatalf("allowed range rejected: %s", got)
		}
	}
}
