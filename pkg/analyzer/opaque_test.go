package analyzer

import (
	"strings"
	"testing"
)

// TestOpaque covers the declaring-package rule of im.Seal and im.Unseal.
func TestOpaque(t *testing.T) {
	const definitions = `type baked struct { weights []float64 }
type Model struct { im.Shared; engine im.Opaque[baked] }
`
	for _, tc := range []struct{ name, source, want string }{
		{"wrap and read", `func Load() *Model { return &Model{engine: im.Seal(&baked{})} }
func Read(m *Model) int { b := im.Unseal(m.engine); return len(b.weights) }`, ""},
		{"copy handle", `func Copy(m *Model) Model { return Model{engine: m.engine} }`, ""},
		{"zero handle", `func Zero() bool { var value im.Opaque[baked]; return im.Unseal(value) == nil }`, ""},
		{"local alias", `type alias = baked
func Read(value im.Opaque[alias]) *alias { return im.Unseal(value) }`, ""},
		{"named slice", `type rows []int
func Read() int { value := rows{1}; return len(*im.Unseal(im.Seal(&value))) }`, ""},
		{"named instance", `type box[T any] struct { value T }
func Read() int { return im.Unseal(im.Seal(&box[int]{value: 1})).value }`, ""},
		{"function values", `func Read(value *baked) *baked {
	seal, unseal := im.Seal[baked], im.Unseal[baked]; return unseal(seal(value)) }`, ""},
		{"generic wrap", `func Wrap[T any](v *T) im.Opaque[T] { return im.Seal(v) }`, "im.Seal requires a type"},
		{"unnamed wrap", `func Wrap(v []int) im.Opaque[[]int] { return im.Seal(&v) }`, "im.Seal requires a type"},
		{"generic read", `func Read[T any](u im.Opaque[T]) *T { return im.Unseal(u) }`, "im.Unseal requires a type"},
		{"generic function value", `func Reader[T any]() func(im.Opaque[T]) *T { return im.Unseal[T] }`,
			"im.Unseal requires a type"},
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
