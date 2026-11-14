package analyzer

import (
	"strings"
	"testing"
)

// TestOpaque covers the declaring-package rule of im.Seal and Opaque.Value.
func TestOpaque(t *testing.T) {
	const definitions = `type baked struct { weights []float64 }
type Model struct { im.Shared; engine im.Opaque[baked] }
`
	for _, tc := range []struct{ name, source, want string }{
		{"wrap and read", `func Load() *Model { return &Model{engine: im.Seal(&baked{})} }
func Read(m *Model) int { b := m.engine.Value(); return len(b.weights) }`, ""},
		{"copy handle", `func Copy(m *Model) Model { return Model{engine: m.engine} }`, ""},
		{"generic wrap", `func Wrap[T any](v *T) im.Opaque[T] { return im.Seal(v) }`, "im.Seal requires a type"},
		{"unnamed wrap", `func Wrap(v []int) im.Opaque[[]int] { return im.Seal(&v) }`, "im.Seal requires a type"},
		{"generic read", `func Read[T any](u im.Opaque[T]) *T { return u.Value() }`, "Opaque.Value requires a type"},
		{"handle write", `func Rebind(m *Model) { m.engine = im.Seal(&baked{}) }`, "writes to immutable objects"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := checkSource(t, definitions+tc.source)
			if (tc.want == "") != (got == "") || !strings.Contains(got, tc.want) {
				t.Fatalf("diagnostics=%q want %q", got, tc.want)
			}
		})
	}
}
