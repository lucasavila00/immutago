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
	})
}
