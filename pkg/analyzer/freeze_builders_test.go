package analyzer

import "testing"

// TestMarkedBuilderOperations checks the public native builder operations with protected elements.
func TestMarkedBuilderOperations(t *testing.T) {
	cases := []string{
		`func f(){var s []V;s=append(s,V{});_=len(s);_=cap(s);clear(s);_=im.FreezeSlice(s)}`,
		`func f(){s:=make([]V,1);t:=[]V{{}};copy(s,t);_=im.FreezeSlice(s);_=im.FreezeSlice(t)}`,
		`func f(){m:=make(map[V]V);m[V{}]=V{};delete(m,V{});clear(m);_=im.FreezeMap(m)}`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, `type V struct{im.Shared;X int};`+source); got != "" {
				t.Fatal(got)
			}
		})
	}
}
