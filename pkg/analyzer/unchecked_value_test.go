package analyzer

import (
	"strings"
	"testing"
)

// TestUncheckedValue covers the declaring-package rule of im.Trust and im.Unwrap.
func TestUncheckedValue(t *testing.T) {
	const definitions = `type baked struct { weights []float64 }
type Model struct { im.Frozen; engine im.UncheckedValue[baked] }
`
	for _, tc := range []struct{ name, source, want string }{
		{"wrap and read", `func Load() *Model { return &Model{engine: im.Trust(&baked{})} }
func Read(m *Model) int { b := im.Unwrap(m.engine); return len(b.weights) }`, ""},
		{"copy handle", `func Copy(m *Model) Model { return Model{engine: m.engine} }`, ""},
		{"zero handle", `func Zero() bool { var value im.UncheckedValue[baked]; return im.Unwrap(value) == nil }`, ""},
		{"local alias", `type alias = baked
func Read(value im.UncheckedValue[alias]) *alias { return im.Unwrap(value) }`, ""},
		{"named slice", `type rows []int
func Read() int { value := rows{1}; return len(*im.Unwrap(im.Trust(&value))) }`, ""},
		{"named instance", `type box[T any] struct { value T }
func Read() int { return im.Unwrap(im.Trust(&box[int]{value: 1})).value }`, ""},
		{"function values", `func Read(value *baked) *baked {
	trust, unwrap := im.Trust[baked], im.Unwrap[baked]; return unwrap(trust(value)) }`, ""},
		{"inferred function value", `var read func(im.UncheckedValue[baked]) *baked = im.Unwrap`, ""},
		{"generic wrap", `func Wrap[T any](v *T) im.UncheckedValue[T] { return im.Trust(v) }`, "im.Trust requires a type"},
		{"unnamed wrap", `func Wrap(v []int) im.UncheckedValue[[]int] { return im.Trust(&v) }`, "im.Trust requires a type"},
		{"generic read", `func Read[T any](u im.UncheckedValue[T]) *T { return im.Unwrap(u) }`, "im.Unwrap requires a type"},
		{"generic function value", `func Reader[T any]() func(im.UncheckedValue[T]) *T { return im.Unwrap[T] }`,
			"im.Unwrap requires a type"},
		{"handle write", `func Rebind(m *Model) { m.engine = im.Trust(&baked{}) }`, "writes to immutable objects"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := checkSource(t, definitions+tc.source)
			if (tc.want == "") != (got == "") || !strings.Contains(got, tc.want) {
				t.Fatalf("diagnostics=%q want %q", got, tc.want)
			}
		})
	}
}

// TestUncheckedReadResults checks the shared result rule and projection extraction.
func TestUncheckedReadResults(t *testing.T) {
	const definitions = `import "github.com/lucasavila00/immutago/pkg/supervise"
 type Model struct{ Values []int }
 type Handle = im.UncheckedValue[Model]
 type Loader struct{ supervise.Service[supervise.NoArgs] }
 `
	for _, source := range []string{
		`func(*Loader) Current()Handle{return Handle{}}`,
		`type Reply struct{im.Local;Model Handle;Models [2]Handle};func(*Loader) Current()Reply{return Reply{}}`,
		`func(*Loader) Current()[2]Handle{return [2]Handle{}}`,
		`func Read(value im.Projection[Handle])Handle{return im.Value(value)}`,
		`func Read(value im.Projection[[2]Handle])[2]Handle{return im.Value(value)}`,
	} {
		t.Run(source, func(t *testing.T) {
			if got := checkSource(t, definitions+source); got != "" {
				t.Fatal(got)
			}
		})
	}
}
