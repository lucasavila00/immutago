package analyzer

import (
	"strings"
	"testing"
)

// TestLiteralTransportRejects checks implicit boxing and callback storage in each literal form.
func TestLiteralTransportRejects(t *testing.T) {
	cases := []struct{ name, source string }{
		{"slice", `func f(v V){_=[]any{v}}`},
		{"indexed slice", `func f(v V){_=[]any{2:v}}`},
		{"array", `func f(v V){_=[1]any{v}}`},
		{"indexed array", `func f(v V){_=[3]any{2:v}}`},
		{"inferred array", `func f(v V){_=[...]any{v}}`},
		{"map key", `func f(v V){_=map[any]int{v:1}}`},
		{"map value", `func f(v V){_=map[int]any{1:v}}`},
		{"keyed struct", `type Holder struct{Value any};func f(v V){_=Holder{Value:v}}`},
		{"unkeyed struct", `type Holder struct{Value any};func f(v V){_=Holder{v}}`},
		{"anonymous struct", `func f(v V){_=struct{Value any}{Value:v}}`},
		{"alias", `type Payload=any;type Holder struct{Value Payload};func f(v V){_=Holder{Value:v}}`},
		{"defined container", `type Values []any;func f(v V){_=Values{v}}`},
		{"nested literal", `func f(v V){_=[][]any{[]any{v}}}`},
		{"elided slice", `func f(v V){_=[][]any{{v}}}`},
		{"elided struct", `type Holder struct{Value any};func f(v V){_=map[int]Holder{1:{Value:v}}}`},
		{"frozen value", `func f(v im.FrozenSlice[int]){_=[]any{v}}`},
		{"callback slice", `func makeValue()V{return V{}};func f(){_=[]func()V{makeValue}}`},
		{"callback field", `type Holder struct{Make func()V};func makeValue()V{return V{}};
   func f(){_=Holder{Make:makeValue}}`},
		{"generic slice", `func f[T ~[]any](v V)T{return T{v}}`},
		{"generic map key", `func f[T ~map[any]int](v V)T{return T{v:1}}`},
		{"generic field", `func f[T ~struct{Value any}](v V)T{return T{Value:v}}`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got := checkSource(t, `type V struct{im.Shared;X int};`+test.source)
			if !strings.Contains(got, "boxing immutable storage") && !strings.Contains(got, "immutable callbacks") &&
				!strings.Contains(got, "protected composite literals require a concrete storage type") {
				t.Fatalf("literal transport accepted: %s", got)
			}
		})
	}
}

// TestLiteralTransportAccepts keeps scalar copies and concrete protected fields available.
func TestLiteralTransportAccepts(t *testing.T) {
	cases := []string{
		`func f(v V){_=[]any{v.X};_=[3]any{2:v.X};_=map[any]any{v.X:v.X}}`,
		`type Holder struct{Value any};func f(v V){_=Holder{Value:v.X};_=Holder{v.X}}`,
		`func f(v V){_=[][]any{{v.X}};_=map[int]struct{Value any}{1:{Value:v.X}}}`,
		`type Owner struct{im.Shared;Child V};func f(v V){_=Owner{Child:v}}`,
		`func f(v V){_=[1]V{v};_=im.FreezeSlice([]V{v});_=im.FreezeMap(map[V]V{v:v})}`,
		`type Holder struct{Make func()int};func scalar()int{return 1};func f(){_=Holder{Make:scalar}}`,
		`func f[T ~[]any](v int)T{return T{v}}`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, `type V struct{im.Shared;X int};`+source); got != "" {
				t.Fatalf("supported literal rejected: %s", got)
			}
		})
	}
}
