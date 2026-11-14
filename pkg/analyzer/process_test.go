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
type ConfigLoader struct {
	proc.Process
	mu      sync.Mutex
	version int
	current im.Owner[Config]
}
func (l *ConfigLoader) Run(ctx context.Context) {
	l.mu.Lock()
	l.version++
	l.mu.Unlock()
	l.current.Store(&Config{Version: l.version})
}
func (l *ConfigLoader) Version() int { l.mu.Lock(); defer l.mu.Unlock(); return l.version }
func (l *ConfigLoader) bump() { l.version++ }
func start(parent *proc.Supervisor) int { loader := proc.Start[ConfigLoader](parent); return loader.Proc().Version() }
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
		{"parenthesized proc", `func f(pid proc.PID[ConfigLoader]) int { return (pid.Proc()).Version() }`},
		{"marked PID field", `type Peers struct { im.Shared; ConfigLoader proc.PID[ConfigLoader] }
func f(p Peers) int { return p.ConfigLoader.Proc().Version() }`},
		{"bound marked PID", `type Peers struct { im.Shared; ConfigLoader proc.PID[ConfigLoader] }
func f(p Peers) int { loader := p.ConfigLoader; return loader.Proc().Version() }`},
		{"PID containers", `var pids []proc.PID[ConfigLoader]; type Plain struct { ConfigLoader proc.PID[ConfigLoader] }`},
		{"shared result", `func (l *ConfigLoader) Latest() *Snapshot { return &Snapshot{Version: l.version} }
func f(pid proc.PID[ConfigLoader]) int { s := pid.Proc().Latest(); return s.Version }`},
		{"immutable parameters", `func (l *ConfigLoader) Merge(limits im.FrozenMap[string, int], peer proc.PID[ConfigLoader],
	value Config, status Status) [2]Status { return [2]Status{status} }`},
		{"process with PID field", `type Cache struct { proc.Process; loader proc.PID[ConfigLoader] }
func (c *Cache) Run(ctx context.Context) { _ = c.loader.Proc().Version() }`},
		{"unexported receiver call", `func (l *ConfigLoader) Bump() { l.bump() }`},
		{"unexported native result", `func (l *ConfigLoader) table() map[string]int { return map[string]int{} }`},
		{"slot load", `func (l *ConfigLoader) Limit() int { c := l.current.Load(); if c == nil { return 0 }
	return c.Version }`},
		{"deferred proc in closure", `func f(pid proc.PID[ConfigLoader]) { defer func() { pid.Proc().Version() }() }`},
		{"promoted receiver call", `type Locked struct { proc.Process; sync.Mutex; n int }
func (l *Locked) Run(context.Context) { l.Lock(); l.n++; l.Unlock() }`},
	})
}

// TestProcessRejected covers rejected process methods, transport, creation, and declarations.
func TestProcessRejected(t *testing.T) {
	checkProcessCases(t, true, [][2]string{
		{"native result", `func (l *ConfigLoader) Table() map[string]int { return nil }`},
		{"callback parameter", `func (l *ConfigLoader) Each(visit func(string)) {}`},
		{"error result", `func (l *ConfigLoader) Load() error { return nil }`},
		{"variadic parameter", `func (l *ConfigLoader) Values(v ...int) {}`},
		{"process result", `func (l *ConfigLoader) self() *ConfigLoader { return l }`},
		{"scoped pointer result", `func (l *ConfigLoader) Current() *Config { return nil }`},
		{"value receiver", `func (l ConfigLoader) Value() int { return 0 }`},
		{"bound proc", `func f(pid proc.PID[ConfigLoader]) { p := pid.Proc(); _ = p }`},
		{"assigned proc", `var saved *ConfigLoader; func f(pid proc.PID[ConfigLoader]) { saved = pid.Proc() }`},
		{"proc field", `func f(pid proc.PID[ConfigLoader]) int { return pid.Proc().version }`},
		{"proc unexported", `func f(pid proc.PID[ConfigLoader]) { pid.Proc().bump() }`},
		{"proc run", `func f(pid proc.PID[ConfigLoader]) { pid.Proc().Run(context.Background()) }`},
		{"proc promoted", `type Locked struct { proc.Process; sync.Mutex }
func (l *Locked) Run(context.Context) {}
func f(pid proc.PID[Locked]) { pid.Proc().Lock() }`},
		{"proc method value", `func f(pid proc.PID[ConfigLoader]) { v := pid.Proc().Version; _ = v }`},
		{"proc value", `func f(pid proc.PID[ConfigLoader]) { get := pid.Proc; _ = get }`},
		{"proc expression", `var get = proc.PID[ConfigLoader].Proc`},
		{"go proc", `func f(pid proc.PID[ConfigLoader]) { go pid.Proc().Version() }`},
		{"defer proc", `func f(pid proc.PID[ConfigLoader]) { defer pid.Proc().Version() }`},
		{"proc boxing", `func f(pid proc.PID[ConfigLoader]) { var x any = pid.Proc(); _ = x }`},
		{"proc argument", `func use(any) {}; func f(pid proc.PID[ConfigLoader]) { use(pid.Proc()) }`},
		{"proc comparison", `func f(pid proc.PID[ConfigLoader]) bool { return pid.Proc() == nil }`},
		{"process copy", `func f(pid proc.PID[ConfigLoader]) { v := *pid.Proc(); _ = v }`},
		{"address literal", `func f() { l := &ConfigLoader{}; _ = l }`},
		{"zero variable", `func f() { var l ConfigLoader; _ = l }`},
		{"new process", `func f() { l := new(ConfigLoader); _ = l }`},
		{"alias", `type L = ConfigLoader`},
		{"defined copy", `type Twin ConfigLoader`},
		{"embedded process", `type Outer struct { ConfigLoader }`},
		{"process parameter", `func f(l *ConfigLoader) {}`},
		{"user generic argument", `func id[T any]() {}; func f() { id[ConfigLoader]() }`},
		{"method expression", `var m = (*ConfigLoader).Version`},
		{"type assertion", `func f(x any) { _, _ = x.(*ConfigLoader) }`},
		{"explicit start arguments", `func f(s *proc.Supervisor) { proc.Start[ConfigLoader, *ConfigLoader](s) }`},
		{"receiver copy", `func (l *ConfigLoader) copy() { other := l; _ = other }`},
		{"receiver argument", `func use(any) {}; func (l *ConfigLoader) send() { use(l) }`},
		{"receiver comparison", `func (l *ConfigLoader) empty() bool { return l == nil }`},
		{"receiver closure", `func (l *ConfigLoader) later() { f := func() { l.version++ }; f() }`},
		{"receiver go", `func (l *ConfigLoader) spawn() { go l.bump() }`},
		{"receiver defer", `func (l *ConfigLoader) flush() { defer l.bump() }`},
		{"receiver method value", `func (l *ConfigLoader) hook() { f := l.bump; f() }`},
		{"receiver run", `func (l *ConfigLoader) again(ctx context.Context) { l.Run(ctx) }`},
		{"receiver dereference", `func (l *ConfigLoader) reset() { *l = ConfigLoader{} }`},
		{"other process unexported", `func (l *ConfigLoader) poke(pid proc.PID[ConfigLoader]) { pid.Proc().bump() }`},
		{"PID of scalar", `var r proc.PID[int]`},
		{"start of plain struct", `type Plain struct{}; func (*Plain) Run(context.Context) {}
func f(s *proc.Supervisor) { proc.Start[Plain](s) }`},
		{"generic PID", `func f[T any](pid proc.PID[T]) {}`},
		{"anonymous process", `var x struct{ proc.Process }`},
		{"generic process", `type G[T any] struct{ proc.Process }`},
		{"named marker field", `type P struct{ M proc.Process }`},
		{"pointer marker", `type P struct{ *proc.Process }`},
		{"marked process", `type P struct{ proc.Process; im.Shared }`},
		{"protected process field", `type P struct{ proc.Process; limits im.FrozenMap[string, int] }`},
	})
}
