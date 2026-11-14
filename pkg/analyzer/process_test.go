package analyzer

import "testing"

const processDefinitions = `import (
	"context"
	"sync"
	"github.com/lucasavila00/immutago/pkg/proc"
)
type Config struct { im.Scoped; Version int; Limits im.FrozenMap[string, int] }
type Snapshot struct { im.Shared; Version int }
type Status int
type Loader struct {
	im.Process
	mu      sync.Mutex
	version int
	current im.Owner[Config]
}
func (l *Loader) Run(ctx context.Context) {
	l.mu.Lock()
	l.version++
	l.mu.Unlock()
	l.current.Store(&Config{Version: l.version})
}
func (l *Loader) Version() int { l.mu.Lock(); defer l.mu.Unlock(); return l.version }
func (l *Loader) bump() { l.version++ }
func start(parent *proc.Supervisor) int { loader := proc.Start[Loader](parent); return loader.Get().Version() }
`

// checkProcessCases runs each source after the process definitions.
func checkProcessCases(t *testing.T, reject bool, cases [][2]string) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc[0], func(t *testing.T) {
			got := checkSource(t, processDefinitions+tc[1])
			if (got != "") != reject {
				t.Fatalf("diagnostics=%q rejected=%v", got, reject)
			}
		})
	}
}

// TestProcessAccepted covers each allowed process form.
func TestProcessAccepted(t *testing.T) {
	checkProcessCases(t, false, [][2]string{
		{"definitions", ``},
		{"parenthesized get", `func f(r proc.Ref[Loader]) int { return (r.Get()).Version() }`},
		{"marked ref field", `type Peers struct { im.Shared; Loader proc.Ref[Loader] }
func f(p Peers) int { return p.Loader.Get().Version() }`},
		{"bound marked ref", `type Peers struct { im.Shared; Loader proc.Ref[Loader] }
func f(p Peers) int { loader := p.Loader; return loader.Get().Version() }`},
		{"ref containers", `var refs []proc.Ref[Loader]; type Plain struct { Loader proc.Ref[Loader] }`},
		{"shared result", `func (l *Loader) Latest() *Snapshot { return &Snapshot{Version: l.version} }
func f(r proc.Ref[Loader]) int { s := r.Get().Latest(); return s.Version }`},
		{"immutable parameters", `func (l *Loader) Merge(limits im.FrozenMap[string, int], peer proc.Ref[Loader],
	value Config, status Status) [2]Status { return [2]Status{status} }`},
		{"process with ref field", `type Cache struct { im.Process; loader proc.Ref[Loader] }
func (c *Cache) Run(ctx context.Context) { _ = c.loader.Get().Version() }`},
		{"unexported receiver call", `func (l *Loader) Bump() { l.bump() }`},
		{"unexported native result", `func (l *Loader) table() map[string]int { return map[string]int{} }`},
		{"slot load", `func (l *Loader) Limit() int { c := l.current.Load(); if c == nil { return 0 }
	return c.Version }`},
		{"deferred get in closure", `func f(r proc.Ref[Loader]) { defer func() { r.Get().Version() }() }`},
		{"promoted receiver call", `type Locked struct { im.Process; sync.Mutex; n int }
func (l *Locked) Run(context.Context) { l.Lock(); l.n++; l.Unlock() }`},
	})
}

// TestProcessRejected covers rejected process methods, transport, creation, and declarations.
func TestProcessRejected(t *testing.T) {
	checkProcessCases(t, true, [][2]string{
		{"native result", `func (l *Loader) Table() map[string]int { return nil }`},
		{"callback parameter", `func (l *Loader) Each(visit func(string)) {}`},
		{"error result", `func (l *Loader) Load() error { return nil }`},
		{"variadic parameter", `func (l *Loader) Values(v ...int) {}`},
		{"process result", `func (l *Loader) self() *Loader { return l }`},
		{"scoped pointer result", `func (l *Loader) Current() *Config { return nil }`},
		{"value receiver", `func (l Loader) Value() int { return 0 }`},
		{"bound get", `func f(r proc.Ref[Loader]) { p := r.Get(); _ = p }`},
		{"assigned get", `var saved *Loader; func f(r proc.Ref[Loader]) { saved = r.Get() }`},
		{"get field", `func f(r proc.Ref[Loader]) int { return r.Get().version }`},
		{"get unexported", `func f(r proc.Ref[Loader]) { r.Get().bump() }`},
		{"get run", `func f(r proc.Ref[Loader]) { r.Get().Run(context.Background()) }`},
		{"get promoted", `type Locked struct { im.Process; sync.Mutex }
func (l *Locked) Run(context.Context) {}
func f(r proc.Ref[Locked]) { r.Get().Lock() }`},
		{"get method value", `func f(r proc.Ref[Loader]) { v := r.Get().Version; _ = v }`},
		{"get value", `func f(r proc.Ref[Loader]) { get := r.Get; _ = get }`},
		{"get expression", `var get = proc.Ref[Loader].Get`},
		{"go get", `func f(r proc.Ref[Loader]) { go r.Get().Version() }`},
		{"defer get", `func f(r proc.Ref[Loader]) { defer r.Get().Version() }`},
		{"get boxing", `func f(r proc.Ref[Loader]) { var x any = r.Get(); _ = x }`},
		{"get argument", `func use(any) {}; func f(r proc.Ref[Loader]) { use(r.Get()) }`},
		{"get comparison", `func f(r proc.Ref[Loader]) bool { return r.Get() == nil }`},
		{"process copy", `func f(r proc.Ref[Loader]) { v := *r.Get(); _ = v }`},
		{"address literal", `func f() { l := &Loader{}; _ = l }`},
		{"zero variable", `func f() { var l Loader; _ = l }`},
		{"new process", `func f() { l := new(Loader); _ = l }`},
		{"alias", `type L = Loader`},
		{"defined copy", `type Twin Loader`},
		{"embedded process", `type Outer struct { Loader }`},
		{"process parameter", `func f(l *Loader) {}`},
		{"user generic argument", `func id[T any]() {}; func f() { id[Loader]() }`},
		{"method expression", `var m = (*Loader).Version`},
		{"type assertion", `func f(x any) { _, _ = x.(*Loader) }`},
		{"explicit start arguments", `func f(s *proc.Supervisor) { proc.Start[Loader, *Loader](s) }`},
		{"receiver copy", `func (l *Loader) copy() { other := l; _ = other }`},
		{"receiver argument", `func use(any) {}; func (l *Loader) send() { use(l) }`},
		{"receiver comparison", `func (l *Loader) empty() bool { return l == nil }`},
		{"receiver closure", `func (l *Loader) later() { f := func() { l.version++ }; f() }`},
		{"receiver go", `func (l *Loader) spawn() { go l.bump() }`},
		{"receiver defer", `func (l *Loader) flush() { defer l.bump() }`},
		{"receiver method value", `func (l *Loader) hook() { f := l.bump; f() }`},
		{"receiver run", `func (l *Loader) again(ctx context.Context) { l.Run(ctx) }`},
		{"receiver dereference", `func (l *Loader) reset() { *l = Loader{} }`},
		{"other process unexported", `func (l *Loader) poke(r proc.Ref[Loader]) { r.Get().bump() }`},
		{"ref of scalar", `var r proc.Ref[int]`},
		{"start of plain struct", `type Plain struct{}; func (*Plain) Run(context.Context) {}
func f(s *proc.Supervisor) { proc.Start[Plain](s) }`},
		{"generic ref", `func f[T any](r proc.Ref[T]) {}`},
		{"anonymous process", `var x struct{ im.Process }`},
		{"generic process", `type G[T any] struct{ im.Process }`},
		{"named marker field", `type P struct{ M im.Process }`},
		{"pointer marker", `type P struct{ *im.Process }`},
		{"marked process", `type P struct{ im.Process; im.Shared }`},
		{"protected process field", `type P struct{ im.Process; limits im.FrozenMap[string, int] }`},
	})
}
