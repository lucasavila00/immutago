package analyzer

import (
	"strings"
	"testing"
)

const copiedMethodTypes = `
type Count int
type Alias = Count
func (c Count) Read() int { return int(c) }
func (c Count) Next() Count { c++; return c }
func (c Count) Pair() (Count, bool) { return c, true }
func (c Count) Add(values ...Count) Count { for _, value := range values { c += value }; return c }
func (c *Count) Inc() { *c++ }
type Score struct { im.Copied; Count Count }
func (s Score) Read() int { return s.Count.Read() }
type Snapshot struct { im.Frozen; Count Count; Alias Alias; Counts [2][2]Count; Score Score }
type Reply struct { im.Local; Count Count; Counts [2][2]Count; Score Score }
`

// TestCopiedScalarReceivers uses the shared Copied proof for every boolean and numeric type.
func TestCopiedScalarReceivers(t *testing.T) {
	for _, scalar := range []string{
		"bool", "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64",
		"uintptr", "float32", "float64", "complex64", "complex128",
	} {
		t.Run(scalar, func(t *testing.T) {
			source := `type Scalar ` + scalar + `;func(s Scalar)Copy()Scalar{return s};
type Frozen struct{im.Frozen;Value Scalar};type Local struct{im.Local;Value Scalar};
func f(frozen *Frozen,local Local)Scalar{_=local.Value.Copy();return frozen.Value.Copy()}`
			if got := checkSource(t, source); got != "" {
				t.Fatal(got)
			}
		})
	}
}

// TestCopiedMethodForms checks copied receiver selection without retaining its protected owner.
func TestCopiedMethodForms(t *testing.T) {
	cases := []struct{ name, source string }{
		{"Frozen value and pointer", `func f(v Snapshot,p *Snapshot)int{_=v.Count.Read();return p.Count.Read()}`},
		{"Local value and pointer", `func f(v Reply,p *Reply)int{_=v.Count.Read();return p.Count.Read()}`},
		{"array and nested record", `func f(v Snapshot,p *Reply){_=v.Counts[0][1].Read();_=p.Score.Count.Read()}`},
		{"alias and parentheses", `func f(v Snapshot){_=((v.Alias).Read)()}`},
		{"zero and literal", `func f(){var v Snapshot;_=v.Count.Read();_=Snapshot{}.Count.Read()}`},
		{"mutate receiver copy", `func f(v Snapshot,p Reply)Count{_=v.Count.Next();return p.Count.Next()}`},
		{"bound copies", `func f(v Snapshot,p Reply)func()int{read:=p.Count.Read;_=read();return v.Count.Read}`},
		{"Copied record bound copies", `func f(v Snapshot,p Reply)func()int{read:=p.Score.Read;_=read();return v.Score.Read}`},
		{"method expressions", `func f(v Snapshot,p Reply){_=Count.Read(v.Count);_=Count.Read(p.Count)}`},
		{"tuple results", `func f(v Snapshot,p Reply)(Count,bool){var n,ok=v.Count.Pair();_=n;_=ok;return p.Count.Pair()}`},
		{"variadic arguments", `func f(v Snapshot)Count{return v.Count.Add(v.Alias,v.Counts[0][0])}`},
		{"generic method", `func(c Count)Map[T any](read func(int)T)T{return read(int(c))};
func read(n int)int{return n};func f(v Snapshot){_=v.Count.Map(read);_=v.Count.Map[int](read)}`},
		{"ordinary copy", `func f(p Reply){count:=p.Count;count.Inc();go count.Inc();defer count.Inc()}`},
		{"Frozen asynchronous reads", `func f(v Snapshot){go v.Count.Read();defer v.Count.Read()}`},
		{"slice builder", `func f()im.FrozenSlice[Count]{values:=[]Count{1};_=values[0].Read();
read:=values[0].Read;frozen:=im.FreezeSlice(values);_=read();return frozen}`},
		{"map builder", `func f()im.FrozenMap[int,Count]{values:=map[int]Count{0:1};_=values[0].Read();
_=Count.Read(values[0]);frozen:=im.FreezeMap(values);return frozen}`},
		{"zero builders", `func f(){var values []Count;var byKey map[int]Count;
if len(values)>0{_=values[0].Read()};_=byKey[0].Read();_=im.FreezeSlice(values);_=im.FreezeMap(byKey)}`},
		{"ordinary scalar slice", `func f(){values:=[]Count{1};values[0].Inc()}`},
		{"direct Local contract", `func(c Count)Reply(value Reply)Reply{return value};
func f(v Snapshot,p Reply)Reply{return v.Count.Reply(p)}`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := checkSource(t, copiedMethodTypes+test.source); got != "" {
				t.Fatal(got)
			}
		})
	}
}

// TestCopiedMethodBoundaries keeps the owner's address, write, and Local statement rules.
func TestCopiedMethodBoundaries(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"Frozen implicit address", `func f(v Snapshot){v.Count.Inc()}`, "receivers must match"},
		{"Local implicit address", `func f(v Reply){v.Count.Inc()}`, "receivers must match"},
		{"array implicit address", `func f(v *Snapshot){v.Counts[0][1].Inc()}`, "receivers must match"},
		{"bound pointer", `func f(v Snapshot){read:=v.Count.Inc;read()}`, "receivers must match"},
		{"explicit address", `func f(v Reply){(&v.Count).Inc()}`, "interior addresses"},
		{"field write", `func f(v Snapshot){v.Count++}`, "writes to immutable"},
		{"Local go", `func f(v Reply){go v.Count.Read()}`, "synchronous"},
		{"Local defer", `func f(v Reply){defer v.Count.Read()}`, "synchronous"},
		{"Local capture", `func f(v Reply)func()int{return func()int{return v.Count.Read()}}`, "closures require"},
		{"Local select", `func f(v Reply,ch chan int){select{case ch<-v.Count.Read():}}`, "select communication"},
		{"Local multiple RHS", `func f(v Reply){a,b:=v.Count.Read(),1;_=a;_=b}`, "multiple assignments"},
		{"builder implicit address", `func f(){values:=[]Count{1};values[0].Inc();_=im.FreezeSlice(values)}`,
			"copying, rebinding, slicing, addressing"},
		{"builder after Freeze", `func f(){values:=[]Count{1};_=im.FreezeSlice(values);_=values[0].Read()}`,
			"consumed"},
		{"builder alias", `func f(){values:=[]Count{1};alias:=values;_=alias[0].Read();_=im.FreezeSlice(values)}`,
			"copying, rebinding, slicing, addressing"},
		{"builder go", `func f(){values:=[]Count{1};go values[0].Read();_=im.FreezeSlice(values)}`, "synchronous"},
		{"builder defer", `func f(){values:=[]Count{1};defer values[0].Read();_=im.FreezeSlice(values)}`, "synchronous"},
		{"Local method value contract", `func(c Count)Reply(value Reply)Reply{return value};
func f(v Snapshot){read:=v.Count.Reply;_=read}`, "method values require"},
		{"non-Copied string", `type Text string;func(t Text)Read()string{return string(t)};
type TextOwner struct{im.Frozen;Text Text};func f(v TextOwner){_=v.Text.Read()}`, "proved marked or Copied receiver"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := checkSource(t, copiedMethodTypes+test.source); !strings.Contains(got, test.want) {
				t.Fatalf("diagnostics = %s; want %s", got, test.want)
			}
		})
	}
}
