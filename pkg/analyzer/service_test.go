package analyzer

import "testing"

const serviceDefinitions = `import (
	"context"
	"sync"
	"github.com/lucasavila00/immutago/pkg/supervise"
)
type Config struct { im.Scoped; Version int; Limits im.FrozenMap[string, int] }
type Snapshot struct { im.Shared; Version int }
type Status int
type LoaderArgs struct { im.Shared; Path string; Limits im.FrozenMap[string, int]; Default *Snapshot }
type ConfigLoader struct {
	supervise.Service[LoaderArgs]
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
type LimiterArgs struct { im.Shared; Loader supervise.Ref[ConfigLoader]; Key string }
type Limiter struct { supervise.Service[LimiterArgs]; used int }
func (l *Limiter) Run(ctx context.Context) { l.used = l.Args().Loader.Current().Limit(l.Args().Key) }
func start(parent *supervise.Group) int {
	loader, _ := supervise.Start[ConfigLoader](parent, LoaderArgs{Path: "config.json"})
	supervise.Start[Limiter](parent, LimiterArgs{Loader: loader, Key: "requests"})
	return loader.Current().Version()
}
`

// checkServiceCases runs each source after the service definitions.
func checkServiceCases(t *testing.T, reject bool, cases [][2]string) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc[0], func(t *testing.T) {
			got := checkSource(t, serviceDefinitions+tc[1])
			if (got != "") != reject {
				t.Fatalf("diagnostics=%q rejected=%v", got, reject)
			}
		})
	}
}

// TestServiceAccepted covers each allowed service form.
func TestServiceAccepted(t *testing.T) {
	checkServiceCases(t, false, [][2]string{
		{"definitions", ``},
		{"entry point alias", `type RunContext = context.Context
type Clock struct { supervise.Service[supervise.NoArgs] }
func (*Clock) Run(RunContext) {}`},
		{"protected arguments", `func f(s *supervise.Group) { limits := im.FreezeMap(map[string]int{"a": 1})
	supervise.Start[ConfigLoader](s, LoaderArgs{Limits: limits, Default: &Snapshot{Version: 1}}) }`},
		{"bound arguments", `func (l *ConfigLoader) Path() string { args := l.Args(); return args.Path }`},
		{"no arguments", `type Cache struct { supervise.Service[supervise.NoArgs]; n int }
func (c *Cache) Run(context.Context) { c.n++ }
func f(s *supervise.Group) (supervise.Ref[Cache], error) { return supervise.Start[Cache](s, supervise.NoArgs{}) }`},
		{"reserved cycle", `type Pong struct { supervise.Service[supervise.Ref[Ping]] }
func (p *Pong) Run(context.Context) { _ = p.Args().Current().Name() }
func (p *Pong) Name() string { return "pong" }
type Ping struct { supervise.Service[supervise.Ref[Pong]] }
func (p *Ping) Run(context.Context) { _ = p.Args().Current().Name() }
func (p *Ping) Name() string { return "ping" }
func f(s *supervise.Group) { pong := supervise.Reserve[Pong](); ping, _ := supervise.Start[Ping](s, pong)
	supervise.StartReserved(s, pong, ping) }`},
		{"parenthesized Current", `func f(ref supervise.Ref[ConfigLoader]) int { return (ref.Current()).Version() }`},
		{"marked Ref field", `type Peers struct { im.Shared; ConfigLoader supervise.Ref[ConfigLoader] }
func f(p Peers) int { return p.ConfigLoader.Current().Version() }`},
		{"bound marked Ref", `type Peers struct { im.Shared; ConfigLoader supervise.Ref[ConfigLoader] }
func f(p Peers) int { loader := p.ConfigLoader; return loader.Current().Version() }`},
		{"Ref containers", `var refs []supervise.Ref[ConfigLoader]
type Plain struct { ConfigLoader supervise.Ref[ConfigLoader] }`},
		{"shared result", `func (l *ConfigLoader) Latest() *Snapshot { return &Snapshot{Version: l.version} }
func f(ref supervise.Ref[ConfigLoader]) int { s := ref.Current().Latest(); return s.Version }`},
		{"immutable parameters", `func (l *ConfigLoader) Merge(
	limits im.FrozenMap[string, int], peer supervise.Ref[ConfigLoader],
	value Config, status Status) [2]Status { return [2]Status{status} }`},
		{"service with Ref field", `type Cache struct {
	supervise.Service[supervise.NoArgs]; loader supervise.Ref[ConfigLoader] }
func (c *Cache) Run(ctx context.Context) { _ = c.loader.Current().Version() }`},
		{"unexported receiver call", `func (l *ConfigLoader) Bump() { l.bump() }`},
		{"unexported native result", `func (l *ConfigLoader) table() map[string]int { return map[string]int{} }`},
		{"slot load", `func (l *ConfigLoader) Level() int { c := l.current.Load(); if c == nil { return 0 }
	return c.Version }`},
		{"deferred Current in closure", `func f(ref supervise.Ref[ConfigLoader]) {
	defer func() { ref.Current().Version() }() }`},
		{"promoted receiver call", `type Locked struct { supervise.Service[supervise.NoArgs]; sync.Mutex; n int }
func (l *Locked) Run(context.Context) { l.Lock(); l.n++; l.Unlock() }`},
	})
}

// TestServiceRejected covers rejected service methods, transport, creation, and declarations.
func TestServiceRejected(t *testing.T) {
	checkServiceCases(t, true, [][2]string{
		{"native result", `func (l *ConfigLoader) Table() map[string]int { return nil }`},
		{"callback parameter", `func (l *ConfigLoader) Each(visit func(string)) {}`},
		{"error result", `func (l *ConfigLoader) Load() error { return nil }`},
		{"variadic parameter", `func (l *ConfigLoader) Values(v ...int) {}`},
		{"service result", `func (l *ConfigLoader) self() *ConfigLoader { return l }`},
		{"scoped pointer result", `func (l *ConfigLoader) Current() *Config { return nil }`},
		{"value receiver", `func (l ConfigLoader) Value() int { return 0 }`},
		{"bound Current", `func f(ref supervise.Ref[ConfigLoader]) { p := ref.Current(); _ = p }`},
		{"assigned Current", `var saved *ConfigLoader; func f(ref supervise.Ref[ConfigLoader]) { saved = ref.Current() }`},
		{"Current field", `func f(ref supervise.Ref[ConfigLoader]) int { return ref.Current().version }`},
		{"Current unexported", `func f(ref supervise.Ref[ConfigLoader]) { ref.Current().bump() }`},
		{"Current run", `func f(ref supervise.Ref[ConfigLoader]) { ref.Current().Run(context.Background()) }`},
		{"Current args", `func f(ref supervise.Ref[ConfigLoader]) { _ = ref.Current().Args() }`},
		{"marker field", `func (l *ConfigLoader) marker() { _ = l.Service }`},
		{"marker assignment", `func (l *ConfigLoader) rebind() { l.Service = supervise.Service[LoaderArgs]{} }`},
		{"marker address", `func (l *ConfigLoader) marker() { _ = &l.Service }`},
		{"args method value", `func (l *ConfigLoader) hook() { f := l.Args; _ = f }`},
		{"missing entry point", `type Idle struct { supervise.Service[supervise.NoArgs] }`},
		{"wrong entry point parameter", `type Wrong struct { supervise.Service[supervise.NoArgs] }
func (*Wrong) Run(int) {}`},
		{"entry point result", `type Wrong struct { supervise.Service[supervise.NoArgs] }
func (*Wrong) Run(context.Context) int { return 0 }`},
		{"entry point value receiver", `type Wrong struct { supervise.Service[supervise.NoArgs] }
func (Wrong) Run(context.Context) {}`},
		{"promoted entry point", `type Runner struct{}
func (*Runner) Run(context.Context) {}
type Wrong struct { supervise.Service[supervise.NoArgs]; Runner }`},
		{"map arguments", `type P struct{ supervise.Service[map[string]int] }`},
		{"error arguments", `type P struct{ supervise.Service[error] }`},
		{"scoped pointer arguments", `type P struct{ supervise.Service[*Config] }`},
		{"two markers", `type Other = supervise.Service[Status]
type P struct{ supervise.Service[supervise.NoArgs]; Other }`},
		{"Current promoted", `type Locked struct { supervise.Service[supervise.NoArgs]; sync.Mutex }
func (l *Locked) Run(context.Context) {}
func f(ref supervise.Ref[Locked]) { ref.Current().Lock() }`},
		{"Current method value", `func f(ref supervise.Ref[ConfigLoader]) { v := ref.Current().Version; _ = v }`},
		{"Current value", `func f(ref supervise.Ref[ConfigLoader]) { get := ref.Current; _ = get }`},
		{"Current expression", `var get = supervise.Ref[ConfigLoader].Current`},
		{"go Current", `func f(ref supervise.Ref[ConfigLoader]) { go ref.Current().Version() }`},
		{"defer Current", `func f(ref supervise.Ref[ConfigLoader]) { defer ref.Current().Version() }`},
		{"Current boxing", `func f(ref supervise.Ref[ConfigLoader]) { var x any = ref.Current(); _ = x }`},
		{"Current argument", `func use(any) {}; func f(ref supervise.Ref[ConfigLoader]) { use(ref.Current()) }`},
		{"Current comparison", `func f(ref supervise.Ref[ConfigLoader]) bool { return ref.Current() == nil }`},
		{"service copy", `func f(ref supervise.Ref[ConfigLoader]) { v := *ref.Current(); _ = v }`},
		{"address literal", `func f() { l := &ConfigLoader{}; _ = l }`},
		{"zero variable", `func f() { var l ConfigLoader; _ = l }`},
		{"new service", `func f() { l := new(ConfigLoader); _ = l }`},
		{"alias", `type L = ConfigLoader`},
		{"defined copy", `type Twin ConfigLoader`},
		{"embedded service", `type Outer struct { ConfigLoader }`},
		{"service parameter", `func f(l *ConfigLoader) {}`},
		{"user generic argument", `func id[T any]() {}; func f() { id[ConfigLoader]() }`},
		{"method expression", `var m = (*ConfigLoader).Version`},
		{"type assertion", `func f(x any) { _, _ = x.(*ConfigLoader) }`},
		{"explicit start arguments", `func f(s *supervise.Group) {
	supervise.Start[ConfigLoader, LoaderArgs](s, LoaderArgs{}) }`},
		{"receiver copy", `func (l *ConfigLoader) copy() { other := l; _ = other }`},
		{"receiver argument", `func use(any) {}; func (l *ConfigLoader) send() { use(l) }`},
		{"receiver comparison", `func (l *ConfigLoader) empty() bool { return l == nil }`},
		{"receiver closure", `func (l *ConfigLoader) later() { f := func() { l.version++ }; f() }`},
		{"receiver go", `func (l *ConfigLoader) spawn() { go l.bump() }`},
		{"receiver defer", `func (l *ConfigLoader) flush() { defer l.bump() }`},
		{"receiver method value", `func (l *ConfigLoader) hook() { f := l.bump; f() }`},
		{"receiver run", `func (l *ConfigLoader) again(ctx context.Context) { l.Run(ctx) }`},
		{"receiver dereference", `func (l *ConfigLoader) reset() { *l = ConfigLoader{} }`},
		{"other service unexported", `func (l *ConfigLoader) poke(ref supervise.Ref[ConfigLoader]) { ref.Current().bump() }`},
		{"Ref of scalar", `var r supervise.Ref[int]`},
		{"reserve of scalar", `var r = supervise.Reserve[int]()`},
		{"start of plain struct", `type Plain struct{ *supervise.Service[supervise.NoArgs] }
func (*Plain) Run(context.Context) {}
func f(s *supervise.Group) { supervise.Start[Plain](s, supervise.NoArgs{}) }`},
		{"generic Ref", `func f[T any](ref supervise.Ref[T]) {}`},
		{"anonymous service", `var x struct{ supervise.Service[supervise.NoArgs] }`},
		{"generic service", `type G[T any] struct{ supervise.Service[T] }`},
		{"named marker field", `type P struct{ M supervise.Service[supervise.NoArgs] }`},
		{"pointer marker", `type P struct{ *supervise.Service[supervise.NoArgs] }`},
		{"marked service", `type P struct{ supervise.Service[supervise.NoArgs]; im.Shared }`},
		{"protected service field", `type P struct{ supervise.Service[supervise.NoArgs]; limits im.FrozenMap[string, int] }`},
	})
}

const poolDefinitions = `type WorkerArgs struct { im.Shared; ID int }
type Worker struct { supervise.Service[WorkerArgs] }
func (*Worker) Run(ctx context.Context) { <-ctx.Done() }
type Pool struct { supervise.Service[int]; sup *supervise.Group; workers []supervise.Ref[Worker] }
var saved *supervise.Group
func use(*supervise.Group) {}
`

// TestServiceChildrenAccepted covers each allowed use of the group of a run.
func TestServiceChildrenAccepted(t *testing.T) {
	checkServiceCases(t, false, [][2]string{
		{"direct start", poolDefinitions + `func (p *Pool) Run(ctx context.Context) {
	for i := range p.Args() { worker, err := supervise.Start[Worker](p.Children(), WorkerArgs{ID: i})
	if err != nil { return }; p.workers = append(p.workers, worker) } }`},
		{"start reserved", poolDefinitions + `func (p *Pool) Run(ctx context.Context) {
	ref := supervise.Reserve[Worker](); supervise.StartReserved(p.Children(), ref, WorkerArgs{}) }`},
		{"nested group", poolDefinitions + `func (p *Pool) Run(ctx context.Context) {
	supervise.Start[Worker](supervise.StartGroup(p.Children()), WorkerArgs{}) }`},
		{"parenthesized", poolDefinitions + `func (p *Pool) Run(ctx context.Context) {
	supervise.Start[Worker]((p.Children()), WorkerArgs{}); supervise.Start[Worker]((p).Children(), WorkerArgs{}) }`},
		{"local", poolDefinitions + `func (p *Pool) Run(ctx context.Context) {
	sup := p.Children()
	for i := range p.Args() { supervise.Start[Worker](sup, WorkerArgs{ID: i}) }
	supervise.StartGroup(sup) }`},
		{"exported method", poolDefinitions + `func (p *Pool) Run(context.Context) {}
func (p *Pool) Grow() { supervise.Start[Worker](p.Children(), WorkerArgs{}) }`},
		{"kept group", poolDefinitions + `func (p *Pool) Run(context.Context) {}
func (p *Pool) Add() { p.sup = supervise.StartGroup(p.Children()); supervise.Start[Worker](p.sup, WorkerArgs{}) }
func (p *Pool) Remove() { <-p.sup.Stop() }`},
		{"method local", poolDefinitions + `func (p *Pool) Run(context.Context) {}
func (p *Pool) grow(n int) { children := p.Children()
	for i := range n { supervise.Start[Worker](children, WorkerArgs{ID: i}) } }`},
		{"user method", poolDefinitions + `func (p *Pool) Run(context.Context) {}
type Own struct { supervise.Service[supervise.NoArgs] }
func (*Own) Run(context.Context) {}
func (*Own) Children() int { return 0 }
func (o *Own) count() int { n := o.Children(); return n }`},
	})
}

// TestServiceChildrenRejected covers rejected uses of the group of a run.
func TestServiceChildrenRejected(t *testing.T) {
	checkServiceCases(t, true, [][2]string{
		{"field", poolDefinitions + `func (p *Pool) Run(context.Context) { p.sup = p.Children() }`},
		{"global", poolDefinitions + `func (p *Pool) Run(context.Context) { saved = p.Children() }`},
		{"helper", poolDefinitions + `func (p *Pool) Run(context.Context) { use(p.Children()) }`},
		{"stop", poolDefinitions + `func (p *Pool) Run(context.Context) { p.Children().Stop() }`},
		{"comparison", poolDefinitions + `func (p *Pool) Run(context.Context) { _ = p.Children() == nil }`},
		{"discard", poolDefinitions + `func (p *Pool) Run(context.Context) { _ = p.Children() }`},
		{"method value", poolDefinitions + `func (p *Pool) Run(context.Context) { f := p.Children; _ = f }`},
		{"go start", poolDefinitions + `func (p *Pool) Run(context.Context) {
	go supervise.Start[Worker](p.Children(), WorkerArgs{}) }`},
		{"defer start", poolDefinitions + `func (p *Pool) Run(context.Context) {
	defer supervise.Start[Worker](p.Children(), WorkerArgs{}) }`},
		{"go call", poolDefinitions + `func (p *Pool) Run(context.Context) { go p.Children() }`},
		{"closure", poolDefinitions + `func (p *Pool) Run(context.Context) {
	func() { supervise.Start[Worker](p.Children(), WorkerArgs{}) }() }`},
		{"return", poolDefinitions + `func (p *Pool) Run(context.Context) {}
func (p *Pool) owned() *supervise.Group { return p.Children() }`},
		{"Current", poolDefinitions + `func (p *Pool) Run(context.Context) {}
func f(ref supervise.Ref[Pool]) { supervise.Start[Worker](ref.Current().Children(), WorkerArgs{}) }`},
		{"marker", poolDefinitions + `func (p *Pool) Run(context.Context) {
	supervise.Start[Worker](p.Service.Children(), WorkerArgs{}) }`},
		{"plain marker", poolDefinitions + `func (p *Pool) Run(context.Context) {}
func f(s *supervise.Group) { var m supervise.Service[supervise.NoArgs]
	supervise.Start[Worker](m.Children(), WorkerArgs{}) }`},
		{"method expression", poolDefinitions + `func (p *Pool) Run(context.Context) {}
var get = (*supervise.Service[int]).Children`},
		{"var local", poolDefinitions + `func (p *Pool) Run(context.Context) {
	var sup = p.Children(); supervise.StartGroup(sup) }`},
		{"reassigned local", poolDefinitions + `func (p *Pool) Run(context.Context) {
	var sup *supervise.Group; sup = p.Children(); supervise.StartGroup(sup) }`},
		{"local field", poolDefinitions + `func (p *Pool) Run(context.Context) { sup := p.Children(); p.sup = sup }`},
		{"local global", poolDefinitions + `func (p *Pool) Run(context.Context) {
	sup := p.Children(); supervise.StartGroup(sup); saved = sup }`},
		{"local helper", poolDefinitions + `func (p *Pool) Run(context.Context) { sup := p.Children(); use(sup) }`},
		{"local copy", poolDefinitions + `func (p *Pool) Run(context.Context) {
	sup := p.Children(); other := sup; supervise.StartGroup(other) }`},
		{"local address", poolDefinitions + `func (p *Pool) Run(context.Context) { sup := p.Children(); _ = &sup }`},
		{"local overwrite", poolDefinitions + `func (p *Pool) Run(context.Context) {
	sup := p.Children(); sup = supervise.NewRoot(nil); supervise.StartGroup(sup) }`},
		{"local stop", poolDefinitions + `func (p *Pool) Run(context.Context) { sup := p.Children(); <-sup.Stop() }`},
		{"local closure", poolDefinitions + `func (p *Pool) Run(context.Context) {
	sup := p.Children(); go func() { supervise.Start[Worker](sup, WorkerArgs{}) }() }`},
		{"local go", poolDefinitions + `func (p *Pool) Run(context.Context) {
	sup := p.Children(); go supervise.StartGroup(sup) }`},
		{"local defer", poolDefinitions + `func (p *Pool) Run(context.Context) {
	sup := p.Children(); defer supervise.StartGroup(sup) }`},
		{"tuple local", poolDefinitions + `func (p *Pool) Run(context.Context) {
	sup, n := p.Children(), 1; _ = n; supervise.StartGroup(sup) }`},
		{"start value", poolDefinitions + `func (p *Pool) Run(context.Context) {
	start := supervise.StartGroup; start(p.Children()) }`},
		{"start argument", poolDefinitions + `type Hold struct { supervise.Service[supervise.NoArgs] }
func (*Hold) Run(context.Context) {}
func (p *Pool) Run(context.Context) {
	supervise.Start[Worker](supervise.NewRoot(nil), WorkerArgs{ID: len([]*supervise.Group{p.Children()})}) }`},
	})
}
