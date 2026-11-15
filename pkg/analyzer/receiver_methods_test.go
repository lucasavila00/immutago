package analyzer

import (
	"strings"
	"testing"
)

const receiverTestTypes = `
type Value struct { im.Frozen; X int; Array [2]int }
type Local struct { im.Local; X int }
func (v Value) Read() int { return v.X }
func (p *Value) ReadPointer() int { if p == nil { return 0 }; return p.X }
func (v Local) Read() int { return v.X }
func (p *Local) ReadPointer() int { if p == nil { return 0 }; return p.X }
`

// TestMarkedMethodForms checks exact receiver types with the existing value and pointer producers.
func TestMarkedMethodForms(t *testing.T) {
	cases := []struct{ name, source string }{
		{"value", `func f(){var v Value;_=v.Read()}`},
		{"pointer", `func f(){p:=&Value{X:1};_=p.ReadPointer()}`},
		{"Local value", `func f(){v:=Local{X:1};_=v.Read()}`},
		{"Local pointer", `func f(){p:=&Local{X:1};_=p.ReadPointer()}`},
		{"nil pointers", `func f(){var p *Value;_=p.ReadPointer();var q *Local=nil;_=q.ReadPointer()}`},
		{"parentheses", `func f(){p:=&Value{};_=((p).ReadPointer)();_=((*p).Read)()}`},
		{"alias", `type Alias=Value;func(a Alias)ReadAlias()int{return a.X};
func f(){a:=Alias{};_=a.ReadAlias();p:=&Alias{};_=p.ReadPointer()}`},
		{"defined struct", `type Copy Value;func(v Copy)ReadCopy()int{return v.X};func f(){v:=Copy{};_=v.ReadCopy()}`},
		{"field", `type Outer struct{im.Frozen;Child Value;P *Value};
func f(o Outer){_=o.Child.Read();_=o.P.ReadPointer()}`},
		{"embedded field", `type Outer struct{im.Frozen;Value};func f(o Outer){_=o.Value.Read()}`},
		{"embedded pointer field", `type Outer struct{im.Frozen;*Value};func f(o Outer){_=o.Value.ReadPointer()}`},
		{"array element", `func f(){a:=[1]Value{{}};_=a[0].Read();b:=[1]*Value{&Value{}};_=b[0].ReadPointer()}`},
		{"builder value", `func f(){s:=[]Value{{}};_=s[0].Read();_=im.FreezeSlice(s)}`},
		{"builder pointer", `func f(){s:=[]*Value{&Value{}};_=s[0].ReadPointer();_=im.FreezeSlice(s)}`},
		{"map pointer", `func f(){m:=map[int]*Value{0:&Value{}};_=m[0].ReadPointer();_=im.FreezeMap(m)}`},
		{"frozen result", `func f(){s:=im.FreezeSlice([]Value{{}});_=s.At(0).Read()}`},
		{"literal", `func f(){_=Value{X:1}.Read();_=(&Value{X:1}).ReadPointer()}`},
		{"new", `func f(){_=new(Value).ReadPointer();v:=Value{};_=new(v).ReadPointer()}`},
		{"call result", `func makeValue()Value{return Value{}};func makePointer()*Value{return &Value{}};
func f(){_=makeValue().Read();_=makePointer().ReadPointer()}`},
		{"Frozen return", `func(p *Value)Self()*Value{return p};func f(){p:=&Value{};q:=p.Self();_=q}`},
		{"Frozen retain", `var saved *Value;func(p *Value)Save(){saved=p};func f(){p:=&Value{};p.Save()}`},
		{"Local copy", `func(p *Local)Copy()Local{return *p};func f(){p:=&Local{};v:=p.Copy();_=v}`},
		{"receiver replacement", `func(v Value)Replace()Value{v=Value{X:2};return v};
func(p *Value)ReplacePointer()*Value{p=&Value{X:2};return p}`},
		{"protected arguments", `func(v Value)ReadOther(p *Local)int{return p.X};
func f(){v:=Value{};p:=&Local{};_=v.ReadOther(p)}`},
		{"scalar variadic", `func(v Value)Sum(xs ...int)int{return v.X+len(xs)};func f(v Value){_=v.Sum(1,2)}`},
		{"tuple forwarding", `func(v Value)Pair()(Value,int){return v,1};func take(Value,int){};
func f(v Value)(Value,int){take(v.Pair());return v.Pair()}`},
		{"scalar copy", `type Count int;func(c *Count)Inc(){*c++};type Outer struct{im.Frozen;Count};
func f(o Outer){c:=o.Count;c.Inc()}`},
		{"ordinary methods", `type Count int;func(c *Count)Inc(){*c++};func(c Count)Read()int{return int(c)};
func f(){var c Count;c.Inc();p:=&c;_=p.Read();later:=p.Inc;later()}`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := checkSource(t, receiverTestTypes+test.source); got != "" {
				t.Fatalf("allowed method form rejected: %s", got)
			}
		})
	}
}

// TestMarkedMethodBoundaries checks complete receiver selection and statement matches.
func TestMarkedMethodBoundaries(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"implicit address", `func f(v Value){_=v.ReadPointer()}`, "declared type exactly"},
		{"implicit dereference", `func f(p *Value){_=p.Read()}`, "declared type exactly"},
		{"field address", `type Outer struct{im.Frozen;Child Value};func f(o *Outer){_=o.Child.ReadPointer()}`,
			"declared type exactly"},
		{"array element address", `func f(){a:=[1]Value{{}};_=a[0].ReadPointer()}`, "declared type exactly"},
		{"builder element address", `func f(){s:=[]Value{{}};_=s[0].ReadPointer();_=im.FreezeSlice(s)}`,
			"declared type exactly"},
		{"promoted value", `type Outer struct{im.Frozen;Value};func f(o Outer){_=o.Read()}`, "promoted immutable"},
		{"promoted pointer", `type Outer struct{im.Frozen;*Value};func f(o Outer){_=o.ReadPointer()}`,
			"promoted immutable"},
		{"nested promotion", `type Inner struct{im.Frozen;Value};type Outer struct{im.Frozen;Inner};
func f(o Outer){_=o.Read()}`, "promoted immutable"},
		{"promoted scalar", `type Count int;func(c *Count)Inc(){*c++};type Outer struct{im.Frozen;Count};
func f(p *Outer){p.Inc()}`, "promoted immutable"},
		{"scalar field pointer", `type Count int;func(c *Count)Inc(){*c++};type Outer struct{im.Frozen;Count};
func f(p *Outer){p.Count.Inc()}`, "concrete non-generic marked receiver"},
		{"scalar field value", `type Count int;func(c Count)Read()int{return int(c)};
type Outer struct{im.Frozen;Count};func f(p *Outer){_=p.Count.Read()}`, "concrete non-generic marked receiver"},
		{"bound value", `func f(v Value){callback:=v.Read;_=callback}`, ""},
		{"bound pointer", `func f(p *Value){callback:=(p.ReadPointer);_=callback}`, "bound immutable methods"},
		{"returned bound method", `func f(p *Value)func()int{return p.ReadPointer}`, "bound immutable methods"},
		{"callback method", `func take(func()int){};func f(p *Value){take(p.ReadPointer)}`, "bound immutable methods"},
		{"value expression", `func f(v Value){_=Value.Read(v)}`, ""},
		{"pointer expression", `func f(p *Value){_=(*Value).ReadPointer(p)}`, "immutable method expressions"},
		{"interface", `func f(v Value){var reader interface{Read()int}=v;_=reader.Read()}`, ""},
		{"dynamic protected argument", `type Reader interface{Read(*Value)int};func f(r Reader,p *Value){_=r.Read(p)}`,
			"dynamic calls with immutable storage"},
		{"go", `func f(p *Value){go p.ReadPointer()}`, "synchronous"},
		{"defer", `func f(p *Value){defer p.ReadPointer()}`, "synchronous"},
		{"capture", `func f(p *Value){callback:=func()int{return p.ReadPointer()};_=callback}`, "closures cannot capture"},
		{"select", `func f(p *Value,ch chan int){select{case ch<-p.ReadPointer():}}`, "statement forms"},

		{"variadic boxing", `func(v Value)Use(...any){};func f(v Value){v.Use(v)}`, ""},
		{"unproved receiver", `type Bad struct{im.Frozen;X []int};func(b Bad)ReadBad()int{return len(b.X)}`,
			"concrete non-generic marked receiver"},
		{"generic receiver", `type Box[T any]struct{im.Frozen;X int};func(b Box[T])Read()int{return b.X}`,
			"user generic declarations"},
		{"named array", `type Counts [2]int;func(c *Counts)Inc(){c[0]++};type Outer struct{im.Frozen;Counts}`,
			"named array"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := checkSource(t, receiverTestTypes+test.source); (test.want == "" && got != "") || !strings.Contains(got, test.want) {
				t.Fatalf("diagnostics = %s; want %s", got, test.want)
			}
		})
	}
}

// TestMarkedMethodBodies applies existing write and escape rules to receiver variables.
func TestMarkedMethodBodies(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"value field", `func(v Value)Change(){v.X=1}`, "writes to immutable"},
		{"pointer field", `func(p *Value)Change(){p.X++}`, "writes to immutable"},
		{"array field", `func(p *Value)Change(){p.Array[0]=1}`, "writes to immutable"},
		{"pointer target", `func(p *Value)Change(){*p=Value{}}`, "writes to immutable"},
		{"receiver address", `func(v Value)Address()*Value{return &v}`, "fresh allocation"},
		{"Local return", `func(p *Local)Self()*Local{return p}`, "returning Local pointers"},
		{"Local nil result", `func(p *Local)Nothing()*Local{return nil}`, "returning Local pointers"},
		{"Local capture", `func(p *Local)Later()func()int{return func()int{return p.X}}`, "closures cannot capture"},
		{"Local stored", `func(p *Local)Save(){s:=[]*Local{p};_=len(s)}`, "Local"},
		{"Local global", `var saved *Local;func(p *Local)Save(){saved=p}`, "package variables"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := checkSource(t, receiverTestTypes+test.source); (test.want == "" && got != "") || !strings.Contains(got, test.want) {
				t.Fatalf("diagnostics = %s; want %s", got, test.want)
			}
		})
	}
}
