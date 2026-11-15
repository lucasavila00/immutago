package analyzer

import (
	"strings"
	"testing"
)

// TestFrozenStringForms checks copied construction, direct reads, and concrete transport.
func TestFrozenStringForms(t *testing.T) {
	cases := []string{
		`func f(text string) im.FrozenString { return im.FreezeString(text[1:]) }`,
		`func f(){var zero im.FrozenString; zero=im.FreezeString("text"); _=zero.Len(); _=zero.At(0)}`,
		`func f(){value:=im.FrozenString{};_=value;type Text=im.FrozenString;var text Text;_=text}`,
		`func f(){text:=im.FreezeString("text");native:=text.Clone();_=native[:1];_=text.Len()}`,
		`func f(){text:="text";_=im.FreezeString(text);_=text[:1]}`,
		`type V struct{im.Shared;Text im.FrozenString};func f()V{return V{Text:im.FreezeString("text")}}`,
		`type V struct{im.Scoped;Text [2]im.FrozenString};func f()V{return V{}}`,
		`func f(){a:=im.FreezeString("same");b:=im.FreezeString("same");_=a==b;_=a!=b}`,
		`func f(){key:=im.FreezeString("key");values:=map[im.FrozenString]int{key:1};frozen:=im.FreezeMap(values);value,ok:=frozen.Get(key);_=value;_=ok}`,
		`func f(){values:=[]im.FrozenString{im.FreezeString("text")};frozen:=im.FreezeSlice(values);for _,text:=range frozen.All(){_=text.Clone()}}`,
		`func read(text im.FrozenString)int{return text.Len()};func f(){_=read(im.FreezeString("text"))}`,
		`func f(){text:=im.FreezeString("text");_=((text.Clone))();_=((im.FreezeString))("text")}`,
		`func f(){type Text string;text:=Text("text");_=im.FreezeString(string(text));_=text[:1]}`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, source); got != "" {
				t.Fatalf("allowed form rejected:\n%s", got)
			}
		})
	}
}

// TestFrozenStringRejectedForms checks the protected source language around each new entry.
func TestFrozenStringRejectedForms(t *testing.T) {
	cases := []struct{ source, want string }{
		{`type V struct{Text im.FrozenString}`, "unmarked struct"},
		{`func f(){text:=im.FreezeString("text");var boxed any=text;_=boxed}`, "boxing immutable storage"},
		{`func f(){text:=im.FreezeString("text");copy:=text.Clone;_=copy()}`, "bound immutable methods"},
		{`func f(){text:=im.FreezeString("text");_=im.FrozenString.Clone(text)}`, "immutable method expressions"},
		{`var freeze=im.FreezeString`, "immutable callbacks"},
		{`func f(){text:=im.FreezeString("text");_=func()string{return text.Clone()}}`, "closures cannot capture"},
		{`func f(){text:=im.FreezeString("text");go text.Clone()}`, "requires synchronous calls"},
		{`func f(){text:=im.FreezeString("text");defer text.Clone()}`, "requires synchronous calls"},
		{`func f(){a,b:=im.FreezeString("a"),1;_=a;_=b}`, "multiple assignments"},
		{`type V struct{im.Shared;Text im.FrozenString};func f(){v:=V{};v.Text=im.FreezeString("text")}`, "writes to immutable objects"},
		{`type Other im.FrozenString;func f(v im.FrozenString)Other{return Other(v)}`, "protected value conversions"},
		{`func f(){var p *im.FrozenString;_=p}`, "concrete marked struct target"},
		{`func f(){text:=im.FreezeString("text");_= &text}`, "concrete marked struct target"},
		{`func identity[T any](v T)T{return v};func f(){_=identity(im.FreezeString("text"))}`, "user generic"},
		{`func read(text ...im.FrozenString){};func f(){read(im.FreezeString("text"))}`, "native slices"},
		{`type Wrapper struct{im.Shared;im.FrozenString};func f(v Wrapper){_=v.Clone()}`, "promoted immutable methods"},
		{`func f(){var values []im.FrozenString;_=values}`, "unsupported owned container"},
	}
	for _, test := range cases {
		t.Run(test.source, func(t *testing.T) {
			if got := checkSource(t, test.source); !strings.Contains(got, test.want) {
				t.Fatalf("diagnostics = %q; want %q", got, test.want)
			}
		})
	}
}

// TestNativeStringStorage checks recursive rejection without restricting ordinary Go strings.
func TestNativeStringStorage(t *testing.T) {
	for _, source := range []string{
		`type V struct{im.Shared;Text string}`,
		`type Text string;type V struct{im.Scoped;Text Text}`,
		`type Text=string;type V struct{im.Shared;Text [0]Text}`,
		`type V struct{im.Shared;Text [2][3]string}`,
		`var value im.FrozenSlice[string]`,
		`var value im.FrozenMap[string,int]`,
		`var value im.FrozenMap[int,string]`,
		`func f(){values:=[]string{"text"};_=im.FreezeSlice(values)}`,
	} {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, source); !strings.Contains(got, "native strings require im.FrozenString") {
				t.Fatalf("native string accepted:\n%s", got)
			}
		})
	}
	ordinary := `var _ im.FrozenString;type Plain struct{Text string};func f(s string) string {
values:=[]string{s};mapping:=map[string]string{s:s};_=values;_=mapping;return s[:1]}`
	if got := checkSource(t, ordinary); got != "" {
		t.Fatalf("ordinary strings rejected:\n%s", got)
	}
}

// TestFrozenStringServiceResults checks the result rule separately from ordinary service data.
func TestFrozenStringServiceResults(t *testing.T) {
	cases := []struct {
		source string
		reject bool
	}{
		{`func(*Counter) Text()im.FrozenString{return im.FreezeString("text")}`, false},
		{`type Text=im.FrozenString;func(*Counter) Text()Text{return Text{}}`, false},
		{`func(*Counter) Text()[2]im.FrozenString{return [2]im.FrozenString{}}`, false},
		{`type Page struct{im.Scoped;Text im.FrozenString};func(*Counter) Text()Page{return Page{}}`, false},
		{`func(*Counter) Text()string{return "text"}`, true},
		{`type Text string;func(*Counter) Text()Text{return "text"}`, true},
		{`type Text=string;func(*Counter) Text()[0]Text{return [0]Text{}}`, true},
		{`type Page struct{im.Scoped;Text string};func(*Counter) Text()Page{return Page{}}`, true},
		{`func(*Counter) text(s string)string{return s[:1]};func(c *Counter) Text(s string)im.FrozenString{return im.FreezeString(c.text(s))}`, false},
	}
	for _, test := range cases {
		t.Run(test.source, func(t *testing.T) {
			got := checkSource(t, serviceDefinitions+test.source)
			if (got != "") != test.reject {
				t.Fatalf("diagnostics = %q; reject = %v", got, test.reject)
			}
		})
	}
}
