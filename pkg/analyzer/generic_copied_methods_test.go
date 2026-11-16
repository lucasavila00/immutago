package analyzer

import (
	"strings"
	"testing"
)

const genericCopiedMethodTypes = `
type Unit struct{}
type Count[U any] int
func (c Count[U]) Read() int { return int(c) }
func (c Count[U]) Next() Count[U] { c++; return c }
func (c Count[U]) Pair() (Count[U], bool) { return c, true }
func (c Count[U]) Add(values ...Count[U]) Count[U] { for _, v := range values { c += v }; return c }
func (c *Count[U]) Inc() { *c++ }
type Score[U any] struct { im.Copied; N int }
func (s Score[U]) Read() int { return s.N }
func (s *Score[U]) Inc() { s.N++ }
type Alias = Count[Unit]
type Snapshot struct { im.Frozen; Count Count[Unit]; Alias Alias; Values [2]Count[Unit]; Score Score[Unit] }
type Reply struct { im.Local; Count Count[Unit]; Values [2]Count[Unit]; Score Score[Unit] }
`

// TestGenericCopiedMethodForms checks the instantiated receiver through the shared type proof.
func TestGenericCopiedMethodForms(t *testing.T) {
	cases := []struct{ name, source string }{
		{"scalar and record", `func f(v Snapshot,p Reply)int{_=p.Count.Read();_=p.Score.Read();
return v.Count.Read()+v.Score.Read()}`},
		{"pointer owners and arrays", `func f(v *Snapshot,p *Reply)int{_=p.Values[0].Read();return v.Values[1].Read()}`},
		{"alias and parentheses", `func f(v Snapshot)int{return ((v.Alias).Read)()}`},
		{"zero and literal", `func f()int{var v Snapshot;_=Snapshot{}.Count.Read();return v.Count.Read()}`},
		{"bound copies", `func f(p Reply)func()int{read:=p.Score.Read;_=read();return p.Count.Read}`},
		{"method expressions", `func f(p Reply)int{return Count[Unit].Read(p.Count)+Score[Unit].Read(p.Score)}`},
		{"tuple and variadic", `func f(v Snapshot)(Count[Unit],bool){_=v.Count.Add(v.Alias,v.Values[0]);
return v.Count.Pair()}`},
		{"method type parameters", `func(c Count[U])Map[T any](read func(int)T)T{return read(int(c))};
func read(n int)int{return n};func f(v Snapshot){_=v.Count.Map(read);_=v.Count.Map[int](read)}`},
		{"ordinary copy", `func f(p Reply){count:=p.Count;count.Inc();go count.Inc();defer count.Inc()}`},
		{"Frozen asynchronous read", `func f(v Snapshot){go v.Count.Read();defer v.Count.Read()}`},
		{"builders", `func f(){values:=[]Count[Unit]{1};read:=values[0].Read;_=im.FreezeSlice(values);_=read();
scores:=map[int]Score[Unit]{0:{N:2}};_=scores[0].Read();_=im.FreezeMap(scores)}`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := checkSource(t, genericCopiedMethodTypes+test.source); got != "" {
				t.Fatal(got)
			}
		})
	}
}

// TestGenericCopiedMethodBoundaries keeps declaration, owner, and builder rules on generic receivers.
func TestGenericCopiedMethodBoundaries(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"scalar pointer", `func f(v Snapshot){v.Count.Inc()}`, "receivers must match"},
		{"record pointer", `func f(p Reply){p.Score.Inc()}`, "receivers must match"},
		{"bound pointer", `func f(p Reply){inc:=p.Values[0].Inc;inc()}`, "receivers must match"},
		{"address", `func f(p Reply){(&p.Count).Inc()}`, "interior addresses"},
		{"write", `func f(v Snapshot){v.Count++}`, "writes to immutable"},
		{"Local go", `func f(p Reply){go p.Count.Read()}`, "synchronous"},
		{"Local defer", `func f(p Reply){defer p.Score.Read()}`, "synchronous"},
		{"Local capture", `func f(p Reply)func()int{return func()int{return p.Count.Read()}}`, "closures require"},
		{"Local multiple RHS", `func f(p Reply){a,b:=p.Count.Read(),1;_=a;_=b}`, "multiple assignments"},
		{"builder pointer", `func f(){values:=[]Count[Unit]{1};values[0].Inc();_=im.FreezeSlice(values)}`,
			"copying, rebinding, slicing, addressing"},
		{"builder after Freeze", `func f(){values:=[]Count[Unit]{1};_=im.FreezeSlice(values);_=values[0].Read()}`,
			"consumed"},
		{"builder alias", `func f(){values:=[]Count[Unit]{1};alias:=values;_=alias[0].Read();_=im.FreezeSlice(values)}`,
			"copying, rebinding, slicing, addressing"},
		{"non-Copied scalar", `type Text[U any]string;func(t Text[U])Read()string{return string(t)};
type TextOwner struct{im.Frozen;Text Text[Unit]};func f(v TextOwner){_=v.Text.Read()}`,
			"proved Frozen, Local, or Copied receiver"},
		{"generic Frozen declaration", `type Bad[U any]struct{im.Frozen;N int};func(b Bad[U])Read()int{return b.N}`,
			"user generic declarations cannot define"},
		{"generic Local declaration", `type Bad[U any]struct{im.Local;N int};func(b Bad[U])Read()int{return b.N}`,
			"user generic declarations cannot define"},
		{"Local type argument", `func f(){var count Count[Reply];_=count.Read()}`, "generic instantiations require"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := checkSource(t, genericCopiedMethodTypes+test.source); !strings.Contains(got, test.want) {
				t.Fatalf("diagnostics = %s; want %s", got, test.want)
			}
		})
	}
}
