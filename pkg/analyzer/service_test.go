package analyzer

import "testing"

const serviceDefinitions = `import (
    "context"
    "github.com/lucasavila00/immutago/pkg/supervise"
)
type Reply struct { im.Local; Count int }
type SnapshotReply struct { im.Frozen; Count int }
type Counter struct { supervise.Service[map[string]int]; count int }
func (*Counter) Run(context.Context) {}
`

// TestServiceResults checks the result category without imposing a service source language.
func TestServiceResults(t *testing.T) {
	cases := []struct {
		name, source string
		reject       bool
	}{
		{"scalar", `func(c *Counter) Count() int { return c.count }`, false},
		{"local value", `func(c *Counter) Read() Reply { return Reply{Count:c.count} }`, false},
		{"frozen value", `func(c *Counter) Read() SnapshotReply { return SnapshotReply{Count:c.count} }`, true},
		{"array", `func(*Counter) Read() [2]int { return [2]int{} }`, false},
		{"frozen", `func(*Counter) Read() im.FrozenSlice[int] { frozen:=im.FreezeSlice([]int{1}); return frozen }`, true},
		{"frozen map",
			`func(*Counter) Read() im.FrozenMap[string,int]{var value im.FrozenMap[string,int];return value}`, true},
		{"frozen array", `func(*Counter) Read() [2]SnapshotReply{return [2]SnapshotReply{}}`, true},
		{"pointer array", `func(*Counter) Read() [2]*SnapshotReply{return [2]*SnapshotReply{}}`, true},
		{"frozen field", `type Page struct{im.Local;Values im.FrozenSlice[int]}
func(*Counter) Read() Page{return Page{}}`, true},
		{"shared field", `type Page struct{im.Local;Value SnapshotReply}
func(*Counter) Read() Page{return Page{}}`, true},
		{"alias", `type Result = Reply;func(*Counter) Read() Result{return Result{}}`, false},
		{"nested array", `func(*Counter) Read() [2][2]Reply{return [2][2]Reply{}}`, false},
		{"defined scalar", `type Count int; func(*Counter) Read() Count{return 1}`, false},
		{"marked array", `func(*Counter) Read() [2]Reply { return [2]Reply{} }`, false},
		{"named array", `type Counts [2]int; func(*Counter) Read() Counts{return Counts{}}`, true},
		{"ordinary receiver", `type Reader struct{count int}
func(r *Reader) Read(p *Reply) Reply{return Reply{Count:p.Count+r.count}}
func use(){r:=&Reader{};p:=&Reply{};_=r.Read(p)}`, false},
		{"interface receiver", `type Reader interface{Read(*Reply) Reply}
func use(r Reader,p *Reply){_=r.Read(p)}`, true},
		{"stored byValue pointer", `type Stored struct{supervise.Service[*Reply]}`, true},
		{"stored Local value", `type Stored struct{supervise.Service[Reply]}`, true},
		{"stored shared pointer", `type Stored struct{supervise.Service[*SnapshotReply]}`, false},
		{"error", `func(c *Counter) Fail() error { return nil }`, false},
		{"error alias", `type Failure = error;func(c *Counter) Fail()(err Failure){return}`, false},
		{"error pair", `func(c *Counter) Read()(Reply,error){return Reply{},nil}`, false},
		{"error scalar pair", `func(c *Counter) Read()(int,error){return 1,nil}`, false},
		{"error promoted", `type Commands struct{};func(Commands) Fail()error{return nil}
 type Outer struct{Counter;Commands}`, false},
		{"error reverse", `func(c *Counter) Read()(error,Reply){return nil,Reply{}}`, false},
		{"two errors", `func(c *Counter) Read()(error,error){return nil,nil}`, false},
		{"three results", `func(c *Counter) Read()(Reply,error,error){return Reply{},nil,nil}`, false},
		{"error defined interface", `type Failure error;func(c *Counter) Fail()Failure{return nil}`, true},
		{"error shadow", `type error interface{Error()string};func(c *Counter) Fail()error{return nil}`, true},
		{"error invalid data", `func(c *Counter) Read()(string,error){return "",nil}`, true},
		{"zero results", `func(c *Counter) Update() { c.count++ }`, false},
		{"promoted zero results", `type Commands struct{}; func(Commands) Pause(){}
type Outer struct{Counter; Commands}`, false},
		{"two results", `func(*Counter) Read()(int,int){return 1,2}`, false},
		{"comma ok", `func(c *Counter) Lookup()(int,bool){return c.count,true}`, false},
		{"named tuple", `func(*Counter) Read()(value Reply,ok bool,err error){return}`, false},
		{"mixed tuple", `func(*Counter) Read()(error,Reply,[2]int,bool,error){return nil,Reply{},[2]int{},true,nil}`, false},
		{"promoted tuple", `type Commands struct{};func(Commands) Read()(Reply,bool){return Reply{},true}
type Outer struct{Counter;Commands}`, false},
		{"invalid first result", `func(*Counter) Read()(string,int,bool){return "",0,false}`, true},
		{"invalid middle result", `func(*Counter) Read()(int,[]int,bool){return 0,nil,false}`, true},
		{"invalid last result", `func(*Counter) Read()(int,bool,SnapshotReply){return 0,false,SnapshotReply{}}`, true},
		{"mutable result", `func(*Counter) Read() map[string]int {return nil}`, true},
		{"pointer result", `func(*Counter) Read() *SnapshotReply {return &SnapshotReply{}}`, true},
		{"byValue pointer", `func(*Counter) Read() *Reply {return nil}`, true},
		{"ordinary dependency", `var dep map[string]int
func use(){dep=map[string]int{}; dep["key"]++}`, false},
		{"private method", `func(*Counter) read() map[string]int {return nil}`, false},
		{"promoted result", `type Mutable struct{}; func(Mutable) Read() []int{return nil}
type Outer struct{Counter; Mutable}`, true},
		{"pointer embedding", `type Other struct{*supervise.Service[int]}
func(*Other) Read() []int{return nil}`, true},
		{"ordinary implementation", `func(c *Counter) Count() int {
self:=c; f:=func(){self.count++}; defer f(); go c.add(); c.add(); return c.count }
func(c *Counter) add(){c.count++}
func use(ref supervise.Ref[Counter]){ref.Borrow().Count()}`, false},
		{"protected write", `func(*Counter) Change(p *Reply) {p.Count++}`, true},
		{"protected capture", `func(*Counter) Read(p *Reply) {f:=func()int{return p.Count};_=f()}`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := checkSource(t, serviceDefinitions+tc.source)
			if (got != "") != tc.reject {
				t.Fatalf("diagnostics=%q rejected=%v", got, tc.reject)
			}
		})
	}
}
