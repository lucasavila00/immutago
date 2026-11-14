package analyzer

import "testing"

// TestDependencyBoundaryRejected checks the wrapper, not the trusted value inside it.
func TestDependencyBoundaryRejected(t *testing.T) {
	checkProcessCases(t, true, [][2]string{
		{"defined wrapper", `type Dep proc.ConcurrentSafe[*int]
type Bad struct { D Dep }
func f(p *int) Bad { return Bad{D: Dep(proc.AssumeConcurrentSafe(p))} }`},
		{"embedded unmarked wrapper", `type Bad struct { proc.ConcurrentSafe[*int] }`},
		{"wrapper pointer field", `type Bad struct { D *proc.ConcurrentSafe[*int] }`},
		{"wrapper array field", `type Bad struct { D [1]proc.ConcurrentSafe[*int] }`},
		{"wrapper slice", `func f() { var ds []proc.ConcurrentSafe[*int]; _ = ds }`},
		{"wrapper array", `func f() { var ds [1]proc.ConcurrentSafe[*int]; _ = ds }`},
		{"interface unwrap", `func f(d proc.ConcurrentSafe[*int]) *int {
	var value interface { Value() *int } = d; return value.Value() }`},
		{"interface argument", `func consume(interface { Value() *int }) {}
func f(d proc.ConcurrentSafe[*int]) { consume(d) }`},
		{"variadic interface", `func consume(...any) {}
func f(d proc.ConcurrentSafe[*int]) { consume(d) }`},
		{"tuple interface", `func pair(d proc.ConcurrentSafe[*int]) (proc.ConcurrentSafe[*int], int) {
	return d, 0 }
func consume(any, int) {}
func f(d proc.ConcurrentSafe[*int]) { consume(pair(d)) }`},
		{"argument pointer", dependencyDefinitions + `func f() { var a *FetcherArgs; _ = a }`},
		{"argument array", dependencyDefinitions + `func f() { var a [1]FetcherArgs; _ = a }`},
		{"global wrapper", `var saved proc.ConcurrentSafe[*int]`},
		{"global arguments", dependencyDefinitions + `var saved FetcherArgs`},
		{"global initializer", `var saved = proc.AssumeConcurrentSafe(new(int))`},
		{"boxed wrapper in trust", `func f(d proc.ConcurrentSafe[*int]) { _ = proc.AssumeConcurrentSafe[any](d) }`},
		{"raw value still checked", `func f() { p := Snapshot{}; _ = proc.AssumeConcurrentSafe(&p) }`},
		{"slice builder", `func f(p *int) {
values := []proc.ConcurrentSafe[*int]{proc.AssumeConcurrentSafe(p)}; _ = len(values) }`},
		{"nil slice builder", `func f() { var values []proc.ConcurrentSafe[*int]; _ = len(values) }`},
		{"argument builder", dependencyDefinitions + `func f() { values := make([]FetcherArgs, 1); _ = len(values) }`},
		{"argument pointer builder", dependencyDefinitions + `func f() { values := []*FetcherArgs{nil}; _ = len(values) }`},
		{"map builder", `func f() { values := map[int]proc.ConcurrentSafe[*int]{}; _ = len(values) }`},
		{"map key builder", `func f() { values := map[proc.ConcurrentSafe[*int]]int{}; _ = len(values) }`},
		{"alias builder", `type Dep = proc.ConcurrentSafe[*int]
func f() { values := make([]Dep, 1); _ = len(values) }`},
	})
}

// TestDependencyBoundaryAccepted keeps aliases and trusted payloads opaque.
func TestDependencyBoundaryAccepted(t *testing.T) {
	checkProcessCases(t, false, [][2]string{
		{"alias", `type Dep = proc.ConcurrentSafe[*int]
type Args struct { im.Shared; D Dep }
func f(p *int) Args { return Args{D: proc.AssumeConcurrentSafe(p)} }`},
		{"protected payload", `type Args struct { im.Shared; D proc.ConcurrentSafe[*Snapshot] }
func f() Args { return Args{D: proc.AssumeConcurrentSafe(&Snapshot{})} }`},
		{"callback payload", `type Args struct { im.Shared; D proc.ConcurrentSafe[func() *Snapshot] }
func source() *Snapshot { return &Snapshot{} }
func f() Args { return Args{D: proc.AssumeConcurrentSafe(source)} }`},
		{"holder metadata", `type Args struct { im.Shared; D proc.ConcurrentSafe[*ConfigLoader] }
func f() Args { return Args{} }`},
		{"slot metadata", `type Args struct { im.Shared; D proc.ConcurrentSafe[im.Owner[int]] }
func f() Args { return Args{} }`},
		{"nested collection metadata", `type Args struct {
	im.Shared; D proc.ConcurrentSafe[im.FrozenSlice[proc.ConcurrentSafe[*int]]] }
func f() Args { return Args{} }`},
		{"explicit callback type", `func source() *Snapshot { return &Snapshot{} }
func f() { d := proc.AssumeConcurrentSafe[func() *Snapshot](source); _ = d }`},
		{"callback literal", `func f() { d := proc.AssumeConcurrentSafe(func() *Snapshot { return &Snapshot{} }); _ = d }`},
		{"protected result", `type Worker struct { proc.Process[proc.ConcurrentSafe[*Snapshot]] }
func (w *Worker) Run(context.Context) { point := w.Args().Value(); _ = point.Version }`},
		{"generic metadata", `type Box[T any] struct { Value T }
type Args struct { im.Shared; D proc.ConcurrentSafe[Box[*Snapshot]] }
func f() Args { return Args{} }`},
		{"ordinary native payload", `func f() {
values := []map[string]int{{}}; d := proc.AssumeConcurrentSafe(values); _ = d }`},
	})
}
