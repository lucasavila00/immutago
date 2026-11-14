package analyzer

import "testing"

const dependencyDefinitions = `
type Logger struct{ prefix string }
func (l *Logger) Print(string) {}
type FetcherArgs struct {
	im.Shared
	BaseURL string
	Log     supervise.ConcurrentSafe[*Logger]
	Dial    supervise.ConcurrentSafe[func(string) *Logger]
}
type Fetcher struct { supervise.Service[FetcherArgs]; log *Logger }
func (f *Fetcher) Run(ctx context.Context) {
	f.Args().Log.Value().Print(f.Args().BaseURL)
	f.log = f.Args().Dial.Value()(f.Args().BaseURL)
}
func startFetcher(s *supervise.Group, log *Logger) (supervise.Ref[Fetcher], error) {
	return supervise.Start[Fetcher](s, FetcherArgs{BaseURL: "x", Log: supervise.AssumeConcurrentSafe(log),
		Dial: supervise.AssumeConcurrentSafe(func(string) *Logger { return log })})
}
`

// TestDependencyAccepted covers each allowed supervise.ConcurrentSafe form.
func TestDependencyAccepted(t *testing.T) {
	checkServiceCases(t, false, [][2]string{
		{"argument fields", dependencyDefinitions},
		{"dependency arguments", `type Worker struct { supervise.Service[supervise.ConcurrentSafe[map[string]int]] }
func (w *Worker) Run(context.Context) { w.Args().Value()["k"]++ }
func f(s *supervise.Group, seen map[string]int) { supervise.Start[Worker](s, supervise.AssumeConcurrentSafe(seen)) }`},
		{"parenthesized value", dependencyDefinitions + `func (f *Fetcher) print() { ((f.Args()).Log).Value().Print("") }`},
		{"trust anywhere", `func f() { d := supervise.AssumeConcurrentSafe(map[string]int{}); _ = d }`},
		{"ordinary function", dependencyDefinitions + `func args(log supervise.ConcurrentSafe[*Logger]) FetcherArgs {
	return FetcherArgs{Log: log} }`},
		{"method parameter", dependencyDefinitions + `func (f *Fetcher) Use(log supervise.ConcurrentSafe[*Logger]) {}`},
		{"local copy of value", dependencyDefinitions + `func (f *Fetcher) print() { log := f.Args().Log.Value()
	log.Print("") }`},
		{"scoped arguments", `type Args struct { im.Scoped; Log supervise.ConcurrentSafe[func()] }
type Worker struct { supervise.Service[Args] }
func (w *Worker) Run(context.Context) { w.Args().Log.Value()() }`},
	})
}

// TestDependencyRejected covers supervise.ConcurrentSafe outside the service argument forms.
func TestDependencyRejected(t *testing.T) {
	checkServiceCases(t, true, [][2]string{
		{"racy map argument", `type ReporterArgs struct { im.Shared; Seen map[string]int }
type Reporter struct { supervise.Service[ReporterArgs] }
func (r *Reporter) Run(context.Context) {}`},
		{"func argument", `type Args struct { im.Shared; Dial func() }`},
		{"method result", dependencyDefinitions + `func (f *Fetcher) Log() supervise.ConcurrentSafe[*Logger] {
	return f.Args().Log }`},
		{"method result args", dependencyDefinitions + `func (f *Fetcher) Copy() FetcherArgs { return f.Args() }`},
		{"owner value", `type State struct { im.Scoped; Log supervise.ConcurrentSafe[*int] }
type Holder struct { supervise.Service[supervise.NoArgs]; state im.Owner[State] }
func (h *Holder) Run(context.Context) {}`},
		{"nested in marked struct", dependencyDefinitions + `type Outer struct { im.Shared; Args FetcherArgs }`},
		{"marked pointer field", dependencyDefinitions + `type Outer struct { im.Shared; Args *FetcherArgs }`},
		{"frozen element", `var deps im.FrozenSlice[supervise.ConcurrentSafe[*int]]`},
		{"frozen args element", dependencyDefinitions + `var all im.FrozenSlice[FetcherArgs]`},
		{"service field", `type Worker struct { supervise.Service[supervise.NoArgs]; log supervise.ConcurrentSafe[*int] }
func (w *Worker) Run(context.Context) {}`},
		{"plain struct field", `type deps struct { log supervise.ConcurrentSafe[*int] }`},
		{"embedded field", `type Args struct { im.Shared; supervise.ConcurrentSafe[*int] }
type Worker struct { supervise.Service[Args] }
func (w *Worker) Run(context.Context) {}`},
		{"array field", `type Args struct { im.Shared; Logs [2]supervise.ConcurrentSafe[*int] }`},
		{"value outside service", `func f(d supervise.ConcurrentSafe[*int]) *int { return d.Value() }`},
		{"value from local args", dependencyDefinitions + `func (f *Fetcher) print() { args := f.Args()
	args.Log.Value().Print("") }`},
		{"value of other args", dependencyDefinitions + `func (f *Fetcher) print(a FetcherArgs) { a.Log.Value().Print("") }`},
		{"value method value", dependencyDefinitions + `func (f *Fetcher) print() { get := f.Args().Log.Value; _ = get }`},
		{"value in go", dependencyDefinitions + `func (f *Fetcher) print() { go f.Args().Log.Value() }`},
		{"value on plain receiver", `type Plain struct { im.Shared; Log supervise.ConcurrentSafe[*int] }
func (p Plain) get() *int { return p.Log.Value() }`},
	})
}
