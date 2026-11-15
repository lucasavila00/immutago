package analyzer

import (
	"strings"
	"testing"
)

const interfaceTestTypes = `
type Scorer interface { im.FrozenInterface; Score([]float32) float64 }
type Tree struct { im.Frozen; Weight float64 }
func (t *Tree) Score(features []float32) float64 { return t.Weight * float64(len(features)) }
type Count struct { im.Frozen; N int }
func (c Count) Score(features []float32) float64 { return float64(c.N) }
`

// TestFrozenInterfaceForms checks each concrete producer and implicit conversion boundary.
func TestFrozenInterfaceForms(t *testing.T) {
	cases := []struct{ name, source string }{
		{"zero and nil", `func f(){var s Scorer; s=nil; var other Scorer=nil; _=s; _=other}`},
		{"value", `func f(){var s Scorer=Count{N:2};_=s.Score(nil)}`},
		{"pointer", `func f(){s:=Scorer(&Tree{Weight:2});_=s.Score([]float32{1})}`},
		{"nil pointer", `func f(){var p *Tree; var s Scorer=p; _=s!=nil}`},
		{"copy", `func f(s Scorer){other:=s;other=s;_=other==s}`},
		{"alias", `type Alias=Scorer;type Pointer=*Tree;func f(p Pointer){var s Alias=(p);_=((s).Score)(nil)}`},
		{"argument", `func read(s Scorer)float64{return s.Score(nil)};func f(){_=read(&Tree{})}`},
		{"return", `func makeScorer()Scorer{return &Tree{Weight:2}};func f(){_=makeScorer().Score(nil)}`},
		{"interface conversion", `type Other interface{Scorer};func f(s Scorer){other:=Other(s);
var copy Scorer=other;_=copy}`},
		{"array", `func f(){values:=[1]Scorer{&Tree{}};_=values[0].Score(nil)}`},
		{"builder", `func f(){values:=[]Scorer{&Tree{}};values=append(values,Count{});
frozen:=im.FreezeSlice(values);_=frozen.At(0).Score(nil)}`},
		{"map", `func f(){values:=map[Scorer]Scorer{Count{}:&Tree{}};frozen:=im.FreezeMap(values);
value,ok:=frozen.Get(Count{});_=value;_=ok}`},
		{"tuple forwarding", `func pair()(*Tree,int){return &Tree{},1};func take(Scorer,int){};func f(){take(pair())}`},
		{"tuple return", `func pair()(*Tree,int){return &Tree{},1};func f()(Scorer,int){return pair()}`},
		{"local result", `type Reader interface{im.FrozenInterface; Read()im.StringProjection};
func(t *Tree)Read()im.StringProjection{return im.ProjectString("text")};
func f(){var r Reader=&Tree{};_=r.Read().Len()}`},
		{"Copied result", `type Summary struct{im.Copied;N int};
type Reader interface{im.FrozenInterface;Summary()Summary};
func(t *Tree)Summary()Summary{return Summary{N:1}};func f(){var r Reader=&Tree{};_=r.Summary()}`},
		{"protected parameters", `type Reader interface{im.FrozenInterface;Read(*Tree)float64};
func(t *Tree)Read(other *Tree)float64{return other.Weight};func f(){var r Reader=&Tree{};_=r.Read(&Tree{})}`},
		{"local parameter", `type Input struct{im.Local;N int};
type Reader interface{im.FrozenInterface;Read(*Input)int};
func(t *Tree)Read(input *Input)int{return input.N};func f(){var r Reader=&Tree{};_=r.Read(&Input{})}`},
		{"receiver retention", `var saved Scorer;func(t *Tree)Save(){saved=t};
type Saver interface{im.FrozenInterface;Save()};func f(){var s Saver=&Tree{};s.Save()}`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := checkSource(t, interfaceTestTypes+test.source); got != "" {
				t.Fatalf("allowed interface form rejected: %s", got)
			}
		})
	}
}

// TestFrozenInterfaceBoundaries checks nearby receiver, transport, and type forms.
func TestFrozenInterfaceBoundaries(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"mutable implementation", `type Bad struct{im.Frozen;Values []int};
func(b *Bad)Score([]float32)float64{return 0};func f(){var s Scorer=&Bad{};_=s}`, "Frozen interface implementation"},
		{"implicit dereference", `func f(){var s Scorer=&Count{};_=s}`, "receivers must match"},
		{"promoted interface call", `type Outer struct{im.Frozen;Scorer};
func f(o Outer){_=o.Score(nil)}`, "promoted immutable methods"},
		{"unmarked nested zero", `type Holder struct{Values [0]Scorer}`, "unmarked struct"},
		{"anonymous", `func f(){var s interface{im.FrozenInterface;Score([]float32)float64}=&Tree{};
_=s}`, "anonymous interfaces"},
		{"local declaration", `func f(){type LocalScorer interface{Scorer};var s LocalScorer;
_=s}`, "package-level declarations"},
		{"promoted implementation", `type Outer struct{im.Frozen;Count};
func f(){var s Scorer=Outer{};_=s}`, "cannot use promoted methods"},
		{"unmarked wrapper", `type Outer struct{Scorer};
func f(s Scorer){var other Scorer=Outer{Scorer:s};_=other}`, "does not embed im.Frozen"},
		{"pointer interface", `func f(){var s *Scorer;_=s}`, "concrete marked struct target"},
		{"typed nil", `func f(){_=Scorer(nil)}`, "protected value conversions"},
		{"ordinary interface", `func f(s Scorer){var other any=s;_=other}`, "boxing immutable"},
		{"assertion", `func f(s Scorer){_=s.(*Tree)}`, "interface assertions"},
		{"type switch", `func f(s Scorer){switch s.(type){case *Tree:}}`, "type switches"},
		{"method expression", `func f(s Scorer){_=Scorer.Score(s,nil)}`, "method expressions"},
		{"bound method", `func f(s Scorer){method:=s.Score;_=method}`, "bound immutable"},
		{"go", `func f(s Scorer){go s.Score(nil)}`, "synchronous"},
		{"defer", `func f(s Scorer){defer s.Score(nil)}`, "synchronous"},
		{"capture", `func f(s Scorer){_=func()float64{return s.Score(nil)}}`, "closures cannot capture"},
		{"callback", `func f(s Scorer,call func(Scorer)){call(s)}`, "callbacks"},
		{"generic", `func identity[T any](v T)T{return v};func f(s Scorer){_=identity(s)}`, "user generic"},
		{"tuple binding", `func pair()(Scorer,int){return &Tree{},1};func f(){s,n:=pair();_=s;
_=n}`, "multiple assignments"},
		{"variadic interface input", `func take(...Scorer){};func f(){take(&Tree{})}`, "native slices"},
		{"variadic boxing", `func take(...any){};func f(s Scorer){take(s)}`, "boxing immutable"},
		{"Copied interface field", `type Bad struct{im.Copied;Scorer Scorer}`, "copied values require"},
		{"raw result", `type Bad interface{im.FrozenInterface;Self()Scorer}`, "Self result"},
		{"string result", `type Bad interface{im.FrozenInterface;Name()string}`, "Name result"},

		{"variadic method", `type Bad interface{im.FrozenInterface;Sum(...int)int}`, "fixed parameters"},
		{"interface parameter", `type Bad interface{im.FrozenInterface;Use(any)}`, "concrete types"},
		{"callback parameter", `type Bad interface{im.FrozenInterface;Use(func())}`, "concrete types"},
		{"unexported method", `type Bad interface{im.FrozenInterface;read()int}`, "must be exported"},
		{"marker payload", `func f(){var s im.FrozenInterface=&Tree{};_=s}`, "markers are annotations"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := checkSource(t, interfaceTestTypes+test.source); !strings.Contains(got, test.want) {
				t.Fatalf("diagnostics = %s; want %s", got, test.want)
			}
		})
	}
}
