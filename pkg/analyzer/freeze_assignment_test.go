package analyzer

import (
	"strings"
	"testing"
)

// TestFreezeAssignment checks standalone Freeze calls into ordinary collection slots.
func TestFreezeAssignment(t *testing.T) {
	const cache = `type Cache struct{Values im.FrozenSlice[int];Index im.FrozenMap[string,int]};`
	cases := map[string]string{
		"interface declaration": `var saved any=im.FreezeSlice([]int{1});
func f(){var boxed any=im.FreezeSlice([]int{2});_=boxed}`,
		"interface assignment": `var saved any;func f(){saved=im.FreezeSlice([]int{1})}`,
		"interface field":      `type Box struct{Value any};func f(b *Box){b.Value=im.FreezeSlice([]int{1})}`,
		"interface map":        `func f(b map[int]any){b[0]=im.FreezeMap(map[int]int{1:2})}`,
		"interface pointer":    `func f(b *any){*b=im.FreezeSlice([]int{1})}`,

		"field":                 `func f(c *Cache){values:=[]int{1};c.Values=im.FreezeSlice(values)}`,
		"map field":             `func f(c *Cache){values:=map[string]int{"a":1};c.Index=im.FreezeMap(values)}`,
		"literal":               `func f(c *Cache){c.Values=im.FreezeSlice([]int{1})}`,
		"array":                 `func f(c *[2]im.FrozenSlice[int]){c[0]=im.FreezeSlice([]int{1})}`,
		"slice":                 `func f(c []im.FrozenSlice[int]){c[0]=im.FreezeSlice([]int{1})}`,
		"map":                   `func f(c map[string]im.FrozenSlice[int]){c["a"]=im.FreezeSlice([]int{1})}`,
		"pointer":               `func f(c *im.FrozenSlice[int]){*c=im.FreezeSlice([]int{1})}`,
		"alias and parentheses": `type Alias=Cache;func f(c *Alias){(c.Values)=(im.FreezeSlice([]int{1}))}`,
		"package":               `var saved im.FrozenSlice[int];func f(){saved=im.FreezeSlice([]int{1})}`,
		"index before consume":  `func f(c []im.FrozenSlice[int]){values:=[]int{1};c[len(values)-1]=im.FreezeSlice(values)}`,
	}
	for name, source := range cases {
		t.Run(name, func(t *testing.T) {
			if got := checkSource(t, cache+source); got != "" {
				t.Fatal(got)
			}
		})
	}
}

// TestFreezeAssignmentBoundaries keeps writes, builder ownership, and statement checks.
func TestFreezeAssignmentBoundaries(t *testing.T) {
	cases := map[string]struct{ source, want string }{
		"protected field": {`type Root struct{im.Frozen;Values im.FrozenSlice[int]};
func f(c *Root){c.Values=im.FreezeSlice([]int{1})}`, "writes to immutable"},
		"alias":     {`func f(c *Cache){values:=[]int{1};alias:=values;_=alias;c.Values=im.FreezeSlice(values)}`, "copying"},
		"consumed":  {`func f(c *Cache){values:=[]int{1};c.Values=im.FreezeSlice(values);values[0]=2}`, "consumed"},
		"parameter": {`func f(c *Cache,values []int){c.Values=im.FreezeSlice(values)}`, "fresh local"},
		"multiple":  {`func f(c *Cache){c.Values,c.Values=im.FreezeSlice([]int{1}),im.FreezeSlice([]int{2})}`, "standalone"},
		"nested":    {`func take(im.FrozenSlice[int]){};func f(){take(im.FreezeSlice([]int{1}))}`, "standalone"},
		"return":    {`func f()im.FrozenSlice[int]{return im.FreezeSlice([]int{1})}`, "standalone"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := checkSource(t, `type Cache struct{Values im.FrozenSlice[int]};`+tc.source)
			if !strings.Contains(got, tc.want) {
				t.Fatalf("diagnostics = %s; want %s", got, tc.want)
			}
		})
	}
}
