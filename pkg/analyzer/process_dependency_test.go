package analyzer

import "testing"

const dependencyDefinitions = `
type Logger struct{ prefix string }
func (l *Logger) Print(string) {}
type FetcherArgs struct {
	im.Shared
	BaseURL string
	Log     proc.Dependency[*Logger]
	Dial    proc.Dependency[func(string) *Logger]
}
type Fetcher struct { proc.Process[FetcherArgs]; log *Logger }
func (f *Fetcher) Run(ctx context.Context) {
	f.Args().Log.Value().Print(f.Args().BaseURL)
	f.log = f.Args().Dial.Value()(f.Args().BaseURL)
}
func startFetcher(s *proc.Supervisor, log *Logger) proc.PID[Fetcher] {
	return proc.Start[Fetcher](s, FetcherArgs{BaseURL: "x", Log: proc.Trust(log),
		Dial: proc.Trust(func(string) *Logger { return log })})
}
`

// TestDependencyAccepted covers each allowed proc.Dependency form.
func TestDependencyAccepted(t *testing.T) {
	checkProcessCases(t, false, [][2]string{
		{"argument fields", dependencyDefinitions},
		{"dependency arguments", `type Worker struct { proc.Process[proc.Dependency[map[string]int]] }
func (w *Worker) Run(context.Context) { w.Args().Value()["k"]++ }
func f(s *proc.Supervisor, seen map[string]int) { proc.Start[Worker](s, proc.Trust(seen)) }`},
		{"parenthesized value", dependencyDefinitions + `func (f *Fetcher) print() { ((f.Args()).Log).Value().Print("") }`},
		{"trust anywhere", `func f() { d := proc.Trust(map[string]int{}); _ = d }`},
		{"ordinary function", dependencyDefinitions + `func args(log proc.Dependency[*Logger]) FetcherArgs {
	return FetcherArgs{Log: log} }`},
		{"local copy of value", dependencyDefinitions + `func (f *Fetcher) print() { log := f.Args().Log.Value()
	log.Print("") }`},
		{"scoped arguments", `type Args struct { im.Scoped; Log proc.Dependency[func()] }
type Worker struct { proc.Process[Args] }
func (w *Worker) Run(context.Context) { w.Args().Log.Value()() }`},
	})
}

// TestDependencyRejected covers proc.Dependency outside the process argument forms.
func TestDependencyRejected(t *testing.T) {
	checkProcessCases(t, true, [][2]string{
		{"racy map argument", `type ReporterArgs struct { im.Shared; Seen map[string]int }
type Reporter struct { proc.Process[ReporterArgs] }
func (r *Reporter) Run(context.Context) {}`},
		{"func argument", `type Args struct { im.Shared; Dial func() }`},
		{"method result", dependencyDefinitions + `func (f *Fetcher) Log() proc.Dependency[*Logger] { return f.Args().Log }`},
		{"method parameter", dependencyDefinitions + `func (f *Fetcher) Use(log proc.Dependency[*Logger]) {}`},
		{"method result args", dependencyDefinitions + `func (f *Fetcher) Copy() FetcherArgs { return f.Args() }`},
		{"owner value", `type State struct { im.Scoped; Log proc.Dependency[*int] }
type Holder struct { proc.Process[proc.None]; state im.Owner[State] }
func (h *Holder) Run(context.Context) {}`},
		{"nested in marked struct", dependencyDefinitions + `type Outer struct { im.Shared; Args FetcherArgs }`},
		{"marked pointer field", dependencyDefinitions + `type Outer struct { im.Shared; Args *FetcherArgs }`},
		{"frozen element", `var deps im.FrozenSlice[proc.Dependency[*int]]`},
		{"frozen args element", dependencyDefinitions + `var all im.FrozenSlice[FetcherArgs]`},
		{"process field", `type Worker struct { proc.Process[proc.None]; log proc.Dependency[*int] }
func (w *Worker) Run(context.Context) {}`},
		{"plain struct field", `type deps struct { log proc.Dependency[*int] }`},
		{"embedded field", `type Args struct { im.Shared; proc.Dependency[*int] }
type Worker struct { proc.Process[Args] }
func (w *Worker) Run(context.Context) {}`},
		{"array field", `type Args struct { im.Shared; Logs [2]proc.Dependency[*int] }`},
		{"protected value", `type Snapshot2 struct { im.Shared; N int }
func f() { d := proc.Trust(&Snapshot2{}); _ = d }`},
		{"value outside process", `func f(d proc.Dependency[*int]) *int { return d.Value() }`},
		{"value from local args", dependencyDefinitions + `func (f *Fetcher) print() { args := f.Args()
	args.Log.Value().Print("") }`},
		{"value of other args", dependencyDefinitions + `func (f *Fetcher) print(a FetcherArgs) { a.Log.Value().Print("") }`},
		{"value method value", dependencyDefinitions + `func (f *Fetcher) print() { get := f.Args().Log.Value; _ = get }`},
		{"value in go", dependencyDefinitions + `func (f *Fetcher) print() { go f.Args().Log.Value() }`},
		{"value on plain receiver", `type Plain struct { im.Shared; Log proc.Dependency[*int] }
func (p Plain) get() *int { return p.Log.Value() }`},
	})
}
