package analyzer

import (
	"strings"
	"testing"
)

const copiedTypes = `
import "github.com/lucasavila00/immutago/pkg/supervise"
var _ supervise.NoArgs
type Score struct { im.Copied; N int; Flags [2]bool }
func (s Score) Read() int { return s.N }
type Pair struct { im.Copied; First Score; Rest [2]Score }
type Holder struct { Score Score; Other int }
type Snapshot struct { im.Frozen; Score Score; Pair Pair }
type Reply struct { im.Local; Score Score }
func readScore(s Score) int { return s.N }
`

// TestCopiedForms checks immutable values in mutable holders and all shared result proofs.
func TestCopiedForms(t *testing.T) {
	cases := []string{
		`var saved Score; func f(){saved=Score{N:1};saved=Score{N:2}}`,
		`type Alias=Score;type Mark=im.Copied;type Other struct{Mark;Value Alias};var value Other`,
		`func f(){type LocalScore struct{im.Copied;N int};_=LocalScore{N:1};_=struct{im.Copied;N int}{N:1}}`,
		`func f(){var h Holder;h.Score=Score{N:1};h.Other++;h.Score=Score{N:2};s:=h.Score;s=Score{N:s.N+1};_=s}`,
		`func f(){h:=&Holder{};h.Score=Score{N:1};h.Other++;v:=*h;*h=Holder{Score:v.Score}}`,
		`func(h *Holder) Set(s Score){h.Score=s;h.Other++};func f(){h:=new(Holder);h.Set(Score{})}`,
		`func f(){values:=[]Score{{N:1}};alias:=values;alias[0]=Score{};values=append(values,Score{});` +
			`_=values[0].Read();_=values[:];clear(values)}`,
		`func f(){values:=[2]Score{};values[0]=Score{N:1};values[1]=values[0];_=values[:]}`,
		`func f(){values:=map[Score]Score{{N:1}:{N:2}};values[Score{}]=Score{};` +
			`value:=values[Score{}];_=value;delete(values,Score{});clear(values)}`,
		`func f(){values:=make([]Score,2);other:=make([]Score,2);copy(values,other);_=cap(values)}`,
		`var cell im.Cell[Score];func f(){cell.Store(Score{N:1});s:=im.Copy(cell.Load());_=s.Read()}`,
		`func f(){s:=im.FreezeSlice([]Score{{N:1}});value:=im.Copy(s.Projection().At(0));` +
			`values:=s.Projection().Clone();alias:=values;alias[0]=value;_=alias}`,
		`func f(){m:=im.FreezeMap(map[Score]Score{{N:1}:{N:2}});values:=m.Projection().Clone();` +
			`values[Score{}]=Score{};_=values}`,
		`func f(value Snapshot)Reply{return Reply{Score:value.Score}}`,
		`func f(value Score){_=readScore(value);_=value.Read()}`,
		`func many(values ...Score){};func f(s Score){many(s);many([]Score{s}...)}`,
		`func f(s Score)Holder{return Holder{Score:s}}`,
		`func f(value Snapshot)Score{return value.Score}`,
		`type Worker struct{supervise.Service[supervise.NoArgs]};func(w *Worker) Last()Pair{return Pair{}}`,
		`type Args struct{Score Score};type Worker struct{supervise.Service[Args]}`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, copiedTypes+source); got != "" {
				t.Fatal(got)
			}
		})
	}
}

// TestCopiedRejectedFields proves declarations before a value reaches an API.
func TestCopiedRejectedFields(t *testing.T) {
	for _, field := range []string{
		"string", "im.FrozenString", "*int", "*Score", "[]int", "map[int]int", "chan int", "func()", "any",
		"[0]string", "[2][0]*Score", "struct{N int}", "Snapshot", "Reply", "im.Projection[int]",
		"im.FrozenSlice[int]", "im.FrozenMap[int,int]", "im.Cell[int]", "im.UncheckedValue[Score]",
		"supervise.Ref[Worker]", "[0]supervise.Ref[Worker]", "im.Copied", "[0]im.Copied",
	} {
		t.Run(field, func(t *testing.T) {
			got := checkSource(t, copiedTypes+`type Worker struct{supervise.Service[supervise.NoArgs]};`+
				`type Bad struct{im.Copied;Value `+field+`}`)
			if !strings.Contains(got, "Copied record:") {
				t.Fatalf("field %s accepted: %s", field, got)
			}
		})
	}
}

// TestCopiedRejectedForms checks writes, aliases, transport, and the protected owner path.
func TestCopiedRejectedForms(t *testing.T) {
	cases := []struct{ source, want string }{
		{`func f(){s:=Score{};s.N=2}`, "writes to immutable objects"},
		{`func f(){s:=Score{};s.Flags[0]=true}`, "writes to immutable objects"},
		{`func f(){p:=Pair{};p.Rest[0]=Score{}}`, "writes to immutable objects"},
		{`func f(){h:=Holder{};h.Score.N++}`, "writes to immutable objects"},
		{`func f(){values:=[]Score{{}};values[0].N++}`, "writes to immutable objects"},
		{`func f(){values:=[2]Score{};values[0].Flags[0]=true}`, "writes to immutable objects"},
		{`func f(v Snapshot){v.Score=Score{}}`, "writes to immutable objects"},
		{`func f(v Reply){v.Score.N++}`, "writes to immutable objects"},
		{`var pointer *Score`, "pointers to Copied records"},
		{`type Pointer=*Score;var pointer Pointer`, "pointers to Copied records"},
		{`type Pointer *Score;var pointer Pointer`, "pointers to Copied records"},
		{`func f(){_=new(Score);_=&Score{}}`, "pointers to Copied records"},
		{`func f(s Score){_=&s}`, "pointers to Copied records"},
		{`func f(h Holder){_=&h.Score}`, "pointers to Copied records"},
		{`func f(s Score){_=&s.N}`, "scalar interior addresses"},
		{`func(s *Score)Change(){s.N++}`, "pointers to Copied records"},
		{`type Embedded struct{Score};func f(s Embedded){_=s.Read()}`, "promoted immutable methods"},
		{`func f(s Score){var value any=s;_=value}`, "boxing immutable storage"},
		{`func f(h Holder){var value any=&h;_=value}`, "boxing immutable storage"},
		{`func identity[T any](v T)T{return v};func f(s Score){_=identity(s)}`, "user generic instantiations"},
		{`func identity[T any](v T)T{return v};func f(s []Score){_=identity(s)}`, "user generic instantiations"},
		{`func f(s Score){_=func()int{return s.N}}`, "closures cannot capture"},
		{`func f(s Score){_=func()int{return len([1]Score{s})}}`, "closures cannot capture"},
		{`func f(s Score){callback:=readScore;_=callback(s)}`, "immutable callbacks"},
		{`func f(s Score){go readScore(s)}`, "synchronous calls"},
		{`func f(s Score){defer readScore(s)}`, "synchronous calls"},
		{`func f(s Score){s,other:=s,s;_,_=s,other}`, "multiple assignments"},
		{`type Worker struct{supervise.Service[supervise.NoArgs]};func(w *Worker)Last()*Score{return nil}`, "service result"},
		{`type Counts [2]int;type Bad struct{im.Copied;N Counts}`, "Copied record:"},
		{`type Bad[T any]struct{im.Copied;N T}`, "Copied record:"},
		{`type Bad struct{Marker im.Copied}`, "markers must be embedded by value"},
		{`type Bad struct{*im.Copied}`, "markers must be embedded by value"},
		{`type Alias=im.Copied;type Bad struct{im.Copied;Alias}`, "exactly one"},
		{`type Bad struct{im.Copied;im.Local}`, "exactly one"},
		{`type Bad struct{im.Copied;im.Frozen}`, "exactly one"},
		{`func f(){values:=[]Score{{}};alias:=values;_=alias;_=im.FreezeSlice(values)}`, "copying"},
		{`func f(){values:=[]Score{{}};_=im.FreezeSlice(values);values[0]=Score{}}`, "consumed"},
	}
	for _, test := range cases {
		t.Run(test.source, func(t *testing.T) {
			if got := checkSource(t, copiedTypes+test.source); !strings.Contains(got, test.want) {
				t.Fatalf("want %q in %s", test.want, got)
			}
		})
	}
}
