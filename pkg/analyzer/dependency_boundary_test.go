package analyzer

import "testing"

// TestDependencyBoundaryRejected checks the wrapper, not the trusted value inside it.
func TestDependencyBoundaryRejected(t *testing.T) {
	checkProcessCases(t, true, [][2]string{
		{"defined wrapper", `type Dep proc.UncheckedDependency[*int]
type Bad struct { D Dep }
func f(p *int) Bad { return Bad{D: Dep(proc.Trust(p))} }`},
		{"embedded unmarked wrapper", `type Bad struct { proc.UncheckedDependency[*int] }`},
		{"wrapper pointer field", `type Bad struct { D *proc.UncheckedDependency[*int] }`},
		{"wrapper array field", `type Bad struct { D [1]proc.UncheckedDependency[*int] }`},
		{"wrapper slice", `func f() { var ds []proc.UncheckedDependency[*int]; _ = ds }`},
		{"wrapper array", `func f() { var ds [1]proc.UncheckedDependency[*int]; _ = ds }`},
		{"interface unwrap", `func f(d proc.UncheckedDependency[*int]) *int {
	var value interface { Value() *int } = d; return value.Value() }`},
		{"interface argument", `func consume(interface { Value() *int }) {}
func f(d proc.UncheckedDependency[*int]) { consume(d) }`},
		{"variadic interface", `func consume(...any) {}
func f(d proc.UncheckedDependency[*int]) { consume(d) }`},
		{"tuple interface", `func pair(d proc.UncheckedDependency[*int]) (proc.UncheckedDependency[*int], int) {
	return d, 0 }
func consume(any, int) {}
func f(d proc.UncheckedDependency[*int]) { consume(pair(d)) }`},
		{"argument pointer", dependencyDefinitions + `func f() { var a *FetcherArgs; _ = a }`},
		{"argument array", dependencyDefinitions + `func f() { var a [1]FetcherArgs; _ = a }`},
		{"global wrapper", `var saved proc.UncheckedDependency[*int]`},
		{"global arguments", dependencyDefinitions + `var saved FetcherArgs`},
		{"global initializer", `var saved = proc.Trust(new(int))`},
		{"boxed wrapper in trust", `func f(d proc.UncheckedDependency[*int]) { _ = proc.Trust[any](d) }`},
		{"raw value still checked", `func f() { p := Snapshot{}; _ = proc.Trust(&p) }`},
	})
}

// TestDependencyBoundaryAccepted keeps aliases and trusted payloads opaque.
func TestDependencyBoundaryAccepted(t *testing.T) {
	checkProcessCases(t, false, [][2]string{
		{"alias", `type Dep = proc.UncheckedDependency[*int]
type Args struct { im.Shared; D Dep }
func f(p *int) Args { return Args{D: proc.Trust(p)} }`},
		{"protected payload", `type Args struct { im.Shared; D proc.UncheckedDependency[*Snapshot] }
func f() Args { return Args{D: proc.Trust(&Snapshot{})} }`},
		{"callback payload", `type Args struct { im.Shared; D proc.UncheckedDependency[func() *Snapshot] }
func source() *Snapshot { return &Snapshot{} }
func f() Args { return Args{D: proc.Trust(source)} }`},
		{"holder metadata", `type Args struct { im.Shared; D proc.UncheckedDependency[*ConfigLoader] }
func f() Args { return Args{} }`},
		{"slot metadata", `type Args struct { im.Shared; D proc.UncheckedDependency[im.Owner[int]] }
func f() Args { return Args{} }`},
		{"nested collection metadata", `type Args struct {
	im.Shared; D proc.UncheckedDependency[im.FrozenSlice[proc.UncheckedDependency[*int]]] }
func f() Args { return Args{} }`},
		{"explicit callback type", `func source() *Snapshot { return &Snapshot{} }
func f() { d := proc.Trust[func() *Snapshot](source); _ = d }`},
		{"callback literal", `func f() { d := proc.Trust(func() *Snapshot { return &Snapshot{} }); _ = d }`},
		{"protected result", `type Worker struct { proc.Process[proc.UncheckedDependency[*Snapshot]] }
func (w *Worker) Run(context.Context) { point := w.Args().Value(); _ = point.Version }`},
		{"generic metadata", `type Box[T any] struct { Value T }
type Args struct { im.Shared; D proc.UncheckedDependency[Box[*Snapshot]] }
func f() Args { return Args{} }`},
	})
}
