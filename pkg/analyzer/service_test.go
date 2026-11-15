package analyzer

import "testing"

const serviceDefinitions = `import (
    "context"
    "github.com/lucasavila00/immutago/pkg/supervise"
)
type Reply struct { im.Scoped; Count int }
type SharedReply struct { im.Shared; Count int }
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
		{"scoped value", `func(c *Counter) Read() Reply { return Reply{Count:c.count} }`, false},
		{"shared value", `func(c *Counter) Read() SharedReply { return SharedReply{Count:c.count} }`, true},
		{"array", `func(*Counter) Read() [2]int { return [2]int{} }`, false},
		{"frozen", `func(*Counter) Read() im.FrozenSlice[int] { frozen:=im.FreezeSlice([]int{1}); return frozen }`, true},
		{"frozen map",
			`func(*Counter) Read() im.FrozenMap[string,int]{var value im.FrozenMap[string,int];return value}`, true},
		{"shared array", `func(*Counter) Read() [2]SharedReply{return [2]SharedReply{}}`, true},
		{"pointer array", `func(*Counter) Read() [2]*SharedReply{return [2]*SharedReply{}}`, true},
		{"frozen field", `type Page struct{im.Scoped;Values im.FrozenSlice[int]}
func(*Counter) Read() Page{return Page{}}`, false},
		{"shared field", `type Page struct{im.Scoped;Value SharedReply}
func(*Counter) Read() Page{return Page{}}`, false},
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
		{"stored scoped pointer", `type Stored struct{supervise.Service[*Reply]}`, true},
		{"stored scoped value", `type Stored struct{supervise.Service[Reply]}`, false},
		{"stored shared pointer", `type Stored struct{supervise.Service[*SharedReply]}`, false},
		{"zero results", `func(c *Counter) Update() { c.count++ }`, false},
		{"promoted zero results", `type Commands struct{}; func(Commands) Pause(){}
type Outer struct{Counter; Commands}`, false},
		{"two results", `func(*Counter) Read()(int,int){return 1,2}`, true},
		{"mutable result", `func(*Counter) Read() map[string]int {return nil}`, true},
		{"pointer result", `func(*Counter) Read() *SharedReply {return &SharedReply{}}`, true},
		{"scoped pointer", `func(*Counter) Read() *Reply {return nil}`, true},
		{"ordinary dependency wrapper", `var dep supervise.ConcurrentSafe[map[string]int]
func use(){dep=supervise.AssumeConcurrentSafe(map[string]int{}); dep.Value()["key"]++}`, false},
		{"private method", `func(*Counter) read() map[string]int {return nil}`, false},
		{"promoted result", `type Mutable struct{}; func(Mutable) Read() []int{return nil}
type Outer struct{Counter; Mutable}`, true},
		{"pointer embedding", `type Other struct{*supervise.Service[int]}
func(*Other) Read() []int{return nil}`, true},
		{"ordinary implementation", `func(c *Counter) Count() int {
self:=c; f:=func(){self.count++}; defer f(); go c.add(); c.add(); return c.count }
func(c *Counter) add(){c.count++}
func use(ref supervise.Ref[Counter]){ref.Current().Count()}`, false},
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
