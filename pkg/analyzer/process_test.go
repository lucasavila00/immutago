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
type LoaderArgs struct { im.Shared; Path string; Limits im.FrozenMap[string, int]; Default *Snapshot }
type ConfigLoader struct {
	proc.Process[LoaderArgs]
	mu      sync.Mutex
	version int
	current im.Owner[Config]
}
func (l *ConfigLoader) Run(ctx context.Context) {
	l.mu.Lock()
	l.version++
	l.mu.Unlock()
	l.current.Store(&Config{Version: l.version, Limits: l.Args().Limits})
}
func (l *ConfigLoader) Version() int { l.mu.Lock(); defer l.mu.Unlock(); return l.version }
func (l *ConfigLoader) bump() { l.version++ }
func (l *ConfigLoader) Limit(key string) int { c := l.current.Load(); if c == nil { return 0 }
	limit, _ := c.Limits.Get(key); return limit }
type LimiterArgs struct { im.Shared; Loader proc.PID[ConfigLoader]; Key string }
type Limiter struct { proc.Process[LimiterArgs]; used int }
func (l *Limiter) Run(ctx context.Context) { l.used = l.Args().Loader.Proc().Limit(l.Args().Key) }
func start(parent *proc.Supervisor) int {
	loader := proc.Start[ConfigLoader](parent, LoaderArgs{Path: "config.json"})
	proc.Start[Limiter](parent, LimiterArgs{Loader: loader, Key: "requests"})
	return loader.Proc().Version()
}
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
		{"entry point alias", `type RunContext = context.Context
type Clock struct { proc.Process[proc.None] }
func (*Clock) Run(RunContext) {}`},
		{"protected arguments", `func f(s *proc.Supervisor) { limits := im.FreezeMap(map[string]int{"a": 1})
	proc.Start[ConfigLoader](s, LoaderArgs{Limits: limits, Default: &Snapshot{Version: 1}}) }`},
		{"bound arguments", `func (l *ConfigLoader) Path() string { args := l.Args(); return args.Path }`},
		{"no arguments", `type Cache struct { proc.Process[proc.None]; n int }
func (c *Cache) Run(context.Context) { c.n++ }
func f(s *proc.Supervisor) proc.PID[Cache] { return proc.Start[Cache](s, proc.None{}) }`},
		{"reserved cycle", `type Pong struct { proc.Process[proc.PID[Ping]] }
func (p *Pong) Run(context.Context) { _ = p.Args().Proc().Name() }
func (p *Pong) Name() string { return "pong" }
type Ping struct { proc.Process[proc.PID[Pong]] }
func (p *Ping) Run(context.Context) { _ = p.Args().Proc().Name() }
func (p *Ping) Name() string { return "ping" }
func f(s *proc.Supervisor) { pong := proc.Reserve[Pong](); ping := proc.Start[Ping](s, pong)
	proc.StartReserved(s, pong, ping) }`},
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
		{"process with PID field", `type Cache struct { proc.Process[proc.None]; loader proc.PID[ConfigLoader] }
func (c *Cache) Run(ctx context.Context) { _ = c.loader.Proc().Version() }`},
		{"unexported receiver call", `func (l *ConfigLoader) Bump() { l.bump() }`},
		{"unexported native result", `func (l *ConfigLoader) table() map[string]int { return map[string]int{} }`},
		{"slot load", `func (l *ConfigLoader) Level() int { c := l.current.Load(); if c == nil { return 0 }
	return c.Version }`},
		{"deferred proc in closure", `func f(pid proc.PID[ConfigLoader]) { defer func() { pid.Proc().Version() }() }`},
		{"promoted receiver call", `type Locked struct { proc.Process[proc.None]; sync.Mutex; n int }
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
		{"proc args", `func f(pid proc.PID[ConfigLoader]) { _ = pid.Proc().Args() }`},
		{"marker field", `func (l *ConfigLoader) marker() { _ = l.Process }`},
		{"marker assignment", `func (l *ConfigLoader) rebind() { l.Process = proc.Process[LoaderArgs]{} }`},
		{"marker address", `func (l *ConfigLoader) marker() { _ = &l.Process }`},
		{"args method value", `func (l *ConfigLoader) hook() { f := l.Args; _ = f }`},
		{"missing entry point", `type Idle struct { proc.Process[proc.None] }`},
		{"wrong entry point parameter", `type Wrong struct { proc.Process[proc.None] }
func (*Wrong) Run(int) {}`},
		{"entry point result", `type Wrong struct { proc.Process[proc.None] }
func (*Wrong) Run(context.Context) int { return 0 }`},
		{"entry point value receiver", `type Wrong struct { proc.Process[proc.None] }
func (Wrong) Run(context.Context) {}`},
		{"promoted entry point", `type Runner struct{}
func (*Runner) Run(context.Context) {}
type Wrong struct { proc.Process[proc.None]; Runner }`},
		{"map arguments", `type P struct{ proc.Process[map[string]int] }`},
		{"error arguments", `type P struct{ proc.Process[error] }`},
		{"scoped pointer arguments", `type P struct{ proc.Process[*Config] }`},
		{"two markers", `type Other = proc.Process[Status]; type P struct{ proc.Process[proc.None]; Other }`},
		{"proc promoted", `type Locked struct { proc.Process[proc.None]; sync.Mutex }
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
		{"explicit start arguments", `func f(s *proc.Supervisor) { proc.Start[ConfigLoader, LoaderArgs](s, LoaderArgs{}) }`},
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
		{"reserve of scalar", `var r = proc.Reserve[int]()`},
		{"start of plain struct", `type Plain struct{ *proc.Process[proc.None] }; func (*Plain) Run(context.Context) {}
func f(s *proc.Supervisor) { proc.Start[Plain](s, proc.None{}) }`},
		{"generic PID", `func f[T any](pid proc.PID[T]) {}`},
		{"anonymous process", `var x struct{ proc.Process[proc.None] }`},
		{"generic process", `type G[T any] struct{ proc.Process[T] }`},
		{"named marker field", `type P struct{ M proc.Process[proc.None] }`},
		{"pointer marker", `type P struct{ *proc.Process[proc.None] }`},
		{"marked process", `type P struct{ proc.Process[proc.None]; im.Shared }`},
		{"protected process field", `type P struct{ proc.Process[proc.None]; limits im.FrozenMap[string, int] }`},
	})
}

const poolDefinitions = `type WorkerArgs struct { im.Shared; ID int }
type Worker struct { proc.Process[WorkerArgs] }
func (*Worker) Run(ctx context.Context) { <-ctx.Done() }
type Pool struct { proc.Process[int]; sup *proc.Supervisor; workers []proc.PID[Worker] }
var saved *proc.Supervisor
func use(*proc.Supervisor) {}
`

// TestProcessSupervisorAccepted covers each allowed use of the supervisor of a run.
func TestProcessSupervisorAccepted(t *testing.T) {
	checkProcessCases(t, false, [][2]string{
		{"direct start", poolDefinitions + `func (p *Pool) Run(ctx context.Context) {
	for i := range p.Args() { p.workers = append(p.workers, proc.Start[Worker](p.Supervisor(), WorkerArgs{ID: i})) } }`},
		{"start reserved", poolDefinitions + `func (p *Pool) Run(ctx context.Context) {
	pid := proc.Reserve[Worker](); proc.StartReserved(p.Supervisor(), pid, WorkerArgs{}) }`},
		{"nested supervisor", poolDefinitions + `func (p *Pool) Run(ctx context.Context) {
	proc.Start[Worker](proc.StartSupervisor(p.Supervisor()), WorkerArgs{}) }`},
		{"parenthesized", poolDefinitions + `func (p *Pool) Run(ctx context.Context) {
	proc.Start[Worker]((p.Supervisor()), WorkerArgs{}); proc.Start[Worker]((p).Supervisor(), WorkerArgs{}) }`},
		{"local", poolDefinitions + `func (p *Pool) Run(ctx context.Context) {
	sup := p.Supervisor()
	for i := range p.Args() { proc.Start[Worker](sup, WorkerArgs{ID: i}) }
	proc.StartSupervisor(sup) }`},
		{"user method", poolDefinitions + `func (p *Pool) Run(context.Context) {}
type Own struct { proc.Process[proc.None] }
func (*Own) Run(context.Context) {}
func (*Own) Supervisor() int { return 0 }
func (o *Own) count() int { n := o.Supervisor(); return n }`},
	})
}

// TestProcessSupervisorRejected covers rejected uses of the supervisor of a run.
func TestProcessSupervisorRejected(t *testing.T) {
	checkProcessCases(t, true, [][2]string{
		{"field", poolDefinitions + `func (p *Pool) Run(context.Context) { p.sup = p.Supervisor() }`},
		{"global", poolDefinitions + `func (p *Pool) Run(context.Context) { saved = p.Supervisor() }`},
		{"helper", poolDefinitions + `func (p *Pool) Run(context.Context) { use(p.Supervisor()) }`},
		{"stop", poolDefinitions + `func (p *Pool) Run(context.Context) { p.Supervisor().Stop() }`},
		{"comparison", poolDefinitions + `func (p *Pool) Run(context.Context) { _ = p.Supervisor() == nil }`},
		{"discard", poolDefinitions + `func (p *Pool) Run(context.Context) { _ = p.Supervisor() }`},
		{"method value", poolDefinitions + `func (p *Pool) Run(context.Context) { f := p.Supervisor; _ = f }`},
		{"go start", poolDefinitions + `func (p *Pool) Run(context.Context) {
	go proc.Start[Worker](p.Supervisor(), WorkerArgs{}) }`},
		{"defer start", poolDefinitions + `func (p *Pool) Run(context.Context) {
	defer proc.Start[Worker](p.Supervisor(), WorkerArgs{}) }`},
		{"go call", poolDefinitions + `func (p *Pool) Run(context.Context) { go p.Supervisor() }`},
		{"closure", poolDefinitions + `func (p *Pool) Run(context.Context) {
	func() { proc.Start[Worker](p.Supervisor(), WorkerArgs{}) }() }`},
		{"outside run", poolDefinitions + `func (p *Pool) Run(context.Context) {}
func (p *Pool) Grow() { proc.Start[Worker](p.Supervisor(), WorkerArgs{}) }`},
		{"return", poolDefinitions + `func (p *Pool) Run(context.Context) {}
func (p *Pool) owned() *proc.Supervisor { return p.Supervisor() }`},
		{"proc", poolDefinitions + `func (p *Pool) Run(context.Context) {}
func f(pid proc.PID[Pool]) { proc.Start[Worker](pid.Proc().Supervisor(), WorkerArgs{}) }`},
		{"marker", poolDefinitions + `func (p *Pool) Run(context.Context) {
	proc.Start[Worker](p.Process.Supervisor(), WorkerArgs{}) }`},
		{"plain marker", poolDefinitions + `func (p *Pool) Run(context.Context) {}
func f(s *proc.Supervisor) { var m proc.Process[proc.None]; proc.Start[Worker](m.Supervisor(), WorkerArgs{}) }`},
		{"method expression", poolDefinitions + `func (p *Pool) Run(context.Context) {}
var get = (*proc.Process[int]).Supervisor`},
		{"var local", poolDefinitions + `func (p *Pool) Run(context.Context) {
	var sup = p.Supervisor(); proc.StartSupervisor(sup) }`},
		{"reassigned local", poolDefinitions + `func (p *Pool) Run(context.Context) {
	var sup *proc.Supervisor; sup = p.Supervisor(); proc.StartSupervisor(sup) }`},
		{"local field", poolDefinitions + `func (p *Pool) Run(context.Context) { sup := p.Supervisor(); p.sup = sup }`},
		{"local global", poolDefinitions + `func (p *Pool) Run(context.Context) {
	sup := p.Supervisor(); proc.StartSupervisor(sup); saved = sup }`},
		{"local helper", poolDefinitions + `func (p *Pool) Run(context.Context) { sup := p.Supervisor(); use(sup) }`},
		{"local copy", poolDefinitions + `func (p *Pool) Run(context.Context) {
	sup := p.Supervisor(); other := sup; proc.StartSupervisor(other) }`},
		{"local address", poolDefinitions + `func (p *Pool) Run(context.Context) { sup := p.Supervisor(); _ = &sup }`},
		{"local overwrite", poolDefinitions + `func (p *Pool) Run(context.Context) {
	sup := p.Supervisor(); sup = proc.NewRoot(); proc.StartSupervisor(sup) }`},
		{"local stop", poolDefinitions + `func (p *Pool) Run(context.Context) { sup := p.Supervisor(); <-sup.Stop() }`},
		{"local closure", poolDefinitions + `func (p *Pool) Run(context.Context) {
	sup := p.Supervisor(); go func() { proc.Start[Worker](sup, WorkerArgs{}) }() }`},
		{"local go", poolDefinitions + `func (p *Pool) Run(context.Context) {
	sup := p.Supervisor(); go proc.StartSupervisor(sup) }`},
		{"local defer", poolDefinitions + `func (p *Pool) Run(context.Context) {
	sup := p.Supervisor(); defer proc.StartSupervisor(sup) }`},
		{"tuple local", poolDefinitions + `func (p *Pool) Run(context.Context) {
	sup, n := p.Supervisor(), 1; _ = n; proc.StartSupervisor(sup) }`},
		{"start value", poolDefinitions + `func (p *Pool) Run(context.Context) {
	start := proc.StartSupervisor; start(p.Supervisor()) }`},
		{"start argument", poolDefinitions + `type Hold struct { proc.Process[proc.None] }
func (*Hold) Run(context.Context) {}
func (p *Pool) Run(context.Context) {
	proc.Start[Worker](proc.NewRoot(), WorkerArgs{ID: len([]*proc.Supervisor{p.Supervisor()})}) }`},
	})
}
