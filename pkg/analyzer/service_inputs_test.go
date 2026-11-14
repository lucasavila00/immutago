package analyzer

import (
	"strings"
	"testing"
)

// TestServiceInputsAccepted checks ordinary Go inputs and direct protected readers.
func TestServiceInputsAccepted(t *testing.T) {
	checkServiceCases(t, false, [][2]string{
		{"interface dependency", `type Generator interface { Next() int }
type Counter struct { n int }
func (g *Counter) Next() int { g.n++; return g.n }
func (l *ConfigLoader) Decide(g Generator) int { return g.Next() }
func f(ref supervise.Ref[ConfigLoader]) int { return ref.Current().Decide(&Counter{}) }`},
		{"native containers and pointer", `func (l *ConfigLoader) Update(m map[string]int, s []int, p *int) {
m["key"]++; s[0]++; (*p)++ }
func f(ref supervise.Ref[ConfigLoader]) { n := 0; ref.Current().Update(map[string]int{}, []int{0}, &n) }`},
		{"callback and channel", `func (l *ConfigLoader) Each(visit func(string), out chan string) {
visit("key"); out <- "key" }
func f(ref supervise.Ref[ConfigLoader]) { ref.Current().Each(func(string) {}, make(chan string, 1)) }`},
		{"variadic input", `func (l *ConfigLoader) Count(values ...int) int { return len(values) }
func f(ref supervise.Ref[ConfigLoader]) { ref.Current().Count(1, 2); values := []int{3, 4}
ref.Current().Count(values...) }`},
		{"retained ordinary input", `var retained map[string]int
func (l *ConfigLoader) Save(values map[string]int) { retained = values }`},
		{"mixed input", `func (l *ConfigLoader) Read(value *Config, visit func(int)) { visit(value.Version) }
func f(ref supervise.Ref[ConfigLoader]) { p := &Config{Version: 1}; ref.Current().Read(p, func(int) {}) }`},
	})
}

// TestServiceInputsKeepProtectedRules checks writes, transport, retention, and result proofs.
func TestServiceInputsKeepProtectedRules(t *testing.T) {
	cases := []struct{ name, source, diagnostic string }{
		{"immutable write", `func (l *ConfigLoader) Change(p *Snapshot) { p.Version = 1 }`,
			"writes to immutable objects"},
		{"interface boxing", `func (l *ConfigLoader) Use(value any) {}
func f(ref supervise.Ref[ConfigLoader]) { ref.Current().Use(&Snapshot{}) }`,
			"boxing immutable storage"},
		{"scoped capture", `func (l *ConfigLoader) Save(p *Config) { f := func() int { return p.Version }; _ = f }`,
			"closures cannot capture immutable storage"},
		{"protected callback", `func (l *ConfigLoader) Each(visit func(*Snapshot)) {}
func visit(*Snapshot) {}
func f(ref supervise.Ref[ConfigLoader]) { ref.Current().Each(visit) }`,
			"immutable callbacks"},
		{"second result", `func (l *ConfigLoader) Copy(input map[string]int) (int, map[string]int) {
return len(input), input }`, "returns map[string]int"},
		{"interface result", `func (l *ConfigLoader) Echo(input any) any { return input }`, "returns any"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := checkSource(t, serviceDefinitions+tc.source)
			if !strings.Contains(got, tc.diagnostic) {
				t.Fatalf("diagnostics = %q, want %q", got, tc.diagnostic)
			}
		})
	}
}
