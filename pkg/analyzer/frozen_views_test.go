package analyzer

import (
	"strings"
	"testing"
)

// TestLocalFrozenForms checks typed local transport and explicit independent copies.
func TestLocalFrozenForms(t *testing.T) {
	cases := []string{
		`func text(s string) im.FrozenByValueString{return im.FreezeByValueString(s)}`,
		`func read(v im.FrozenByValueString)string{return v.Clone()}
func f(){v:=im.FreezeByValueString("text");_=read(v);_=v.Len();_=v.At(0)}`,
		`type R struct{im.ByValue;Text im.FrozenByValueString}
func pass(r R)R{return r};func f(){r:=R{Text:im.FreezeByValueString("text")};other:=pass(r);_=other.Text.Clone()}`,
		`func f(){owned:=im.FreezeString("text");view:=owned.ByValue();_=view.Clone()}`,
		`func f(){owned:=im.FreezeSlice([]int{1});view:=owned.ByValue();_=view.At(0);for _,n:=range view.All(){_=n}}`,
		`func f(){view:=im.FreezeByValueSlice([]int{1});copy:=view.Clone();copy[0]=2;_=im.FreezeSlice(copy)}`,
		`func f(){key:=im.FreezeString("key");owned:=im.FreezeMap(map[im.FrozenString]int{key:1});view:=owned.ByValue()
value,ok:=view.Get(key);_=value;_=ok;copy:=view.Clone();_=im.FreezeMap(copy)}`,
		`func f(){view:=im.FreezeByValueMap(map[int]int{1:2});for key,value:=range view.All(){_=key;_=value}}`,
		`func f(){text:=im.FreezeByValueString("text");values:=[]im.FrozenByValueString{text}
values[0]=text;view:=im.FreezeByValueSlice(values);item:=view.At(0);_=item.Clone()}`,
		`func pass(values [2]im.FrozenByValueString)[2]im.FrozenByValueString{return values}
func f(){values:=[2]im.FrozenByValueString{};_=pass(values)}`,
		`type State struct{im.ByValue;Values im.FrozenSlice[int]};var slot im.Owner[State]`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, source); got != "" {
				t.Fatalf("allowed local form rejected:\n%s", got)
			}
		})
	}
}

// TestLocalFrozenStorage rejects each persistent route, including enclosing values.
func TestLocalFrozenStorage(t *testing.T) {
	cases := []struct{ source, want string }{
		{`var text im.FrozenByValueString`, "package variables"},
		{`type R struct{im.ByValue;Text im.FrozenByValueString};var saved R`, "package variables"},
		{`var values [0]im.FrozenByValueString`, "package variables"},
		{`type R struct{im.Shared;Text im.FrozenByValueString}`, "Shared fields"},
		{`type R struct{im.ByValue;Text im.FrozenByValueString};type S struct{im.Shared;R R}`, "Shared fields"},
		{`type R struct{im.ByValue;Text im.FrozenByValueString};var slot im.Owner[R]`, "Owner cannot store"},
		{`type R struct{im.ByValue;Text im.FrozenByValueString};var value im.UncheckedValue[R]`, "unchecked storage"},
		{`func f(){text:=im.FreezeByValueString("text");values:=[]im.FrozenByValueString{text};_=im.FreezeSlice(values)}`,
			"shareable frozen storage"},
		{`func f(){view:=im.FreezeByValueSlice([]int{1});iterator:=view.All();_=iterator}`, "direct range"},
		{`func f(){view:=im.FreezeByValueMap(map[int]int{1:2});iterator:=view.All();_=iterator}`, "direct range"},
		{`func f(){view:=im.FreezeByValueString("text");f:=func(){_=view.Len()};f()}`, "closures cannot capture"},
		{`func f(){view:=im.FreezeByValueString("text");var value any=view;_=value}`, "boxing immutable storage"},
		{`func f(){view:=im.FreezeByValueString("text");go view.Len()}`, "synchronous"},
		{`func f(){values:=[]int{1};_=im.FreezeByValueSlice(values);_=values[0]}`, "consumed owned container"},
		{`func f(){values:=[]int{1};alias:=values;_=im.FreezeByValueSlice(values);_=alias}`, "unsupported owned container"},
	}
	for _, test := range cases {
		t.Run(test.source, func(t *testing.T) {
			if got := checkSource(t, test.source); !strings.Contains(got, test.want) {
				t.Fatalf("diagnostics = %q; want %q", got, test.want)
			}
		})
	}
}

// TestLocalFrozenExtraction checks element, key, and Clone retention boundaries.
func TestLocalFrozenExtraction(t *testing.T) {
	cases := []struct{ source, want string }{
		{`func f(){values:=im.FreezeSlice([]im.FrozenString{im.FreezeString("text")});_=values.ByValue()}`,
			"local frozen reads cannot expose"},
		{`type Item struct{im.Shared;N int};func f(){values:=im.FreezeSlice([]*Item{&Item{}});_=values.ByValue()}`,
			"local frozen reads cannot expose"},
		{`type Item struct{im.ByValue;Text im.FrozenString};var view im.FrozenByValueSlice[Item]`,
			"local frozen reads cannot expose"},
		{`func f(){key:=im.FreezeString("key");view:=im.FreezeByValueMap(map[im.FrozenString]int{key:1})
for key,value:=range view.All(){_=key;_=value}}`, "local map iteration cannot expose"},
		{`func f(){text:=im.FreezeByValueString("text");view:=im.FreezeByValueSlice([]im.FrozenByValueString{text})
copy:=view.Clone();_=copy}`, "Clone cannot copy nested local views"},
	}
	for _, test := range cases {
		t.Run(test.source, func(t *testing.T) {
			if got := checkSource(t, test.source); !strings.Contains(got, test.want) {
				t.Fatalf("diagnostics = %q; want %q", got, test.want)
			}
		})
	}
}

// TestLocalFrozenResults rejects hidden shared references and permits typed view results.
func TestLocalFrozenResults(t *testing.T) {
	cases := []struct {
		source string
		reject bool
	}{
		{`func(*Counter) Text()im.FrozenByValueString{return im.FreezeByValueString("text")}`, false},
		{`func(*Counter) Text()im.FrozenString{return im.FreezeString("text")}`, true},
		{`type R struct{im.ByValue;Values im.FrozenSlice[int]};func(*Counter) Read()R{return R{}}`, true},
		{`type R struct{im.ByValue;Values im.FrozenByValueSlice[int]};func(*Counter) Read()R{return R{}}`, false},
		{`type R struct{im.ByValue;P *SharedReply};func(*Counter) Read()R{return R{}}`, true},
		{`type R struct{im.ByValue;Text im.FrozenByValueString};type S struct{supervise.Service[R]}`, true},
	}
	for _, test := range cases {
		t.Run(test.source, func(t *testing.T) {
			if got := checkSource(t, serviceDefinitions+test.source); (got != "") != test.reject {
				t.Fatalf("diagnostics = %q; reject = %v", got, test.reject)
			}
		})
	}
}
