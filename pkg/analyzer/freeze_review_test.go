package analyzer

import (
	"strings"
	"testing"
)

// TestFreezeReviewRejectsIndirectEntries checks that function transport cannot hide the freeze transition.
func TestFreezeReviewRejectsIndirectEntries(t *testing.T) {
	cases := []struct{ name, source string }{
		{"aggregate", `type Holder struct{Freeze func([]int)im.FrozenSlice[int]}
   func f(){h:=Holder{Freeze:im.FreezeSlice[int]};s:=[]int{1};frozen:=h.Freeze(s);s[0]=2;_=frozen}`},
		{"slice callback", `func f(){calls:=[]func([]int)im.FrozenSlice[int]{im.FreezeSlice[int]};
   s:=[]int{1};frozen:=calls[0](s);s[0]=2;_=frozen}`},
		{"map aggregate", `type Holder struct{Freeze func(map[int]int)im.FrozenMap[int,int]}
   func f(){h:=Holder{Freeze:im.FreezeMap[int,int]};m:=map[int]int{1:2};frozen:=h.Freeze(m);m[1]=3;_=frozen}`},
		{"local function", `func f(){freeze:=im.FreezeSlice[int];s:=[]int{1};_=freeze(s);s[0]=2}`},
		{"named function type", `type Freezer func([]int)im.FrozenSlice[int]
   func f(){var freeze Freezer=im.FreezeSlice[int];s:=[]int{1};_=freeze(s);s[0]=2}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := checkSource(t, tc.source)
			if !strings.Contains(got, "freeze functions must be called directly") &&
				!strings.Contains(got, "immutable callbacks are unsupported") {
				t.Fatalf("indirect freeze entry was not rejected at its source: %s", got)
			}
		})
	}
}

// TestFreezeReviewRejectsNativeAliases checks alias-producing operations before the ownership transition.
func TestFreezeReviewRejectsNativeAliases(t *testing.T) {
	cases := []struct{ name, source string }{
		{"element pointer", `func f(){s:=[]int{1};p:=&s[0];frozen:=im.FreezeSlice(s);*p=2;_=frozen}`},
		{"slice view", `func f(){s:=[]int{1};alias:=s[:];frozen:=im.FreezeSlice(s);alias[0]=2;_=frozen}`},
		{"named method", `type Raw []int;func(r Raw)Alias()[]int{return r}
   func f(){s:=Raw{1};alias:=s.Alias();frozen:=im.FreezeSlice(s);alias[0]=2;_=frozen}`},
		{"closure", `func f(){s:=[]int{1};write:=func(){s[0]=2};frozen:=im.FreezeSlice(s);write();_=frozen}`},
		{"aggregate handle", `type Holder struct{S []int}
   func f(){s:=[]int{1};h:=Holder{S:s};frozen:=im.FreezeSlice(s);h.S[0]=2;_=frozen}`},
		{"map handle", `func f(){m:=map[int]int{1:2};alias:=m;frozen:=im.FreezeMap(m);alias[1]=3;_=frozen}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := checkSource(t, tc.source); got == "" {
				t.Fatal("native storage alias survived freezing")
			}
		})
	}
}

// TestFreezeReviewDirectSyntax checks equivalent direct syntax and independent clones.
func TestFreezeReviewDirectSyntax(t *testing.T) {
	cases := []string{
		`func f(){s:=[]int{1};_=((im.FreezeSlice[int]))(s)}`,
		`func f(){m:=map[int]int{1:2};_=((im.FreezeMap[int,int]))(m)}`,
		`func f(){frozen:=im.FreezeSlice([]int{1});s:=frozen.Clone();s[0]=2;_=im.FreezeSlice(s)}`,
		`func f(){frozen:=im.FreezeMap(map[int]int{1:2});m:=frozen.Clone();m[1]=3;_=im.FreezeMap(m)}`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, source); got != "" {
				t.Fatalf("direct freeze or independent clone rejected: %s", got)
			}
		})
	}
}

// TestFreezeReviewDelayedWrites checks writes whose handle was evaluated before Freeze consumed it.
func TestFreezeReviewDelayedWrites(t *testing.T) {
	cases := []struct{ name, source string }{
		{"slice assignment", `func f(){s:=[]int{1};var frozen im.FrozenSlice[int];s[0],frozen=2,im.FreezeSlice(s);_=frozen}`},
		{"map assignment", `func f(){m:=map[int]int{1:2};var frozen im.FrozenMap[int,int];
   m[1],frozen=3,im.FreezeMap(m);_=frozen}`},
		{"copy argument", `func value(im.FrozenSlice[int])int{return 9}
   func f(){s:=[]int{1};copy(s,[]int{value(im.FreezeSlice(s))})}`},
		{"delete key", `func key(im.FrozenMap[int,int])int{return 1}
   func f(){m:=map[int]int{1:2};delete(m,key(im.FreezeMap(m)))}`},
		{"indexed increment", `func index(im.FrozenSlice[int])int{return 0}
   func f(){s:=[]int{1};s[index(im.FreezeSlice(s))]++}`},
		{"deferred clear", `func f(){s:=[]int{1};defer clear(s);_=im.FreezeSlice(s)}`},
		{"concurrent clear", `func f(){s:=[]int{1};go clear(s);_=im.FreezeSlice(s)}`},
		{"deferred delete", `func f(){m:=map[int]int{1:2};defer delete(m,1);_=im.FreezeMap(m)}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := checkSource(t, tc.source); got == "" {
				t.Fatal("delayed native write survived freezing")
			}
		})
	}
}
