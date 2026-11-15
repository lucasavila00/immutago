package analyzer

import "testing"

// TestCopiedMutable checks ordinary Go forms on value copies and their holders.
func TestCopiedMutable(t *testing.T) {
	cases := []string{
		`func f(){s:=Score{};s.N++;s.Flags[0]=true;p:=Pair{};p.Rest[0].N=2}`,
		`func f(){h:=Holder{};h.Score.N++;values:=[]Score{{}};values[0].N++;` +
			`array:=[2]Score{};array[0].Flags[0]=true}`,
		`type Pointer=*Score;type Defined *Score;var zero **Score;` +
			`func f(s Score,h Holder){p:=&s;p.N++;_= &s.N;_= &h.Score;_=new(Score);_=&Score{};var d Defined=p;_=d}`,
		`func(s *Score)Change(){s.N++};func f(h *Holder){h.Score.Change();p:=&h.Score;p.Change()}`,
		`type Number int;func(n *Number)Inc(){*n++};type Count struct{im.Copied;N Number};` +
			`func f(v Count){v.N.Inc()}`,
		`type Embedded struct{Score};func f(s Embedded){_=s.Read()}`,
		`func f(s Score){var value any=s;_=value;var pointer any=&s;_=pointer}`,
		`func f(h Holder){var value any=&h;_=value}`,
		`func identity[T any](v T)T{return v};func f(s Score){_=identity(s);_=identity([]Score{s});_=identity(&s)}`,
		`func f(s Score){_=func()int{return s.N};_=func()int{return len([1]Score{s})}}`,
		`func f(s Score){callback:=readScore;_=callback(s);method:=s.Read;_=method();_=Score.Read(s)}`,
		`func f(s Score){go readScore(s);defer readScore(s);s,other:=s,s;_,_=s,other}`,
		`func (s Score)Values()(int,bool){return s.N,s.Flags[0]};` +
			`func f(s Score){v,ok:=s.Values();_,_=v,ok}`,
		`type request struct{id string;limits Score;a,b int};func(r *request)release(){};` +
			`func f(r *request,lookup func(string)(float64,bool)){defer r.release();go r.release();` +
			`x,y:=r.a,r.b;v,ok:=lookup(r.id);r.a,r.b=1,2;_,_,_,_=x,y,v,ok}`,
		`func f(s Score){values:=make(chan Score,1);values<-s;v,ok:=<-values;_,_=v,ok}`,
		`func f(s Score){switch v:=any(&s).(type){case *Score:v.N++;case Score:v.N++}}`,
		`func f(v Snapshot){s:=v.Score;s.N++;p:=&s;p.Flags[0]=true}`,
		`func f(v Reply){s:=v.Score;s.N++}`,
		`func f(v Snapshot){_=v.Score.Read();_=v.Pair.Rest[0].Read()}`,
		`func f(v Snapshot,callback func(Score)int){_=callback(v.Score);var boxed any=v.Score;_=boxed}`,
		`func identity[T any](v T)T{return v};func f(v Snapshot){_=identity(v.Score);_=Holder{Score:v.Score}}`,
		`func f(){values:=[]Score{{}};values[0].N++;_=im.FreezeSlice(values)}`,
		`func f(){values:=im.FreezeSlice([]Score{{}});copy:=values.At(0);copy.N++}`,
		`func f(){values:=im.FreezeSlice([]Score{{}});clone:=values.Projection().Clone();clone[0].N++}`,
		`var cell im.Cell[Score];func f(){s:=cell.LoadShared();s.N++}`,
		`func(s *Score)Change(){s.N++};func f(v Snapshot){s:=v.Score;s.Change()}`,
		`func(s *Score)Change(){s.N++};func f(v Reply){s:=v.Score;s.Change()}`,
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, copiedTypes+source); got != "" {
				t.Fatal(got)
			}
		})
	}
}
