package analyzer

import (
	"strings"
	"testing"
)

// TestGenericAliasBoundaries checks that type erasure cannot hide conflicting loans or unmarked owners.
func TestGenericAliasBoundaries(t *testing.T) {
	const prefix = `type Value struct { im.Immut; N int }
 `
	cases := []struct{ name, source, want string }{
		{"wrapper", `type Box[T any] struct { V T }
   func box[T any](v T) any { return Box[T]{V:v} }; var _ = box(Value{})`, "unresolved generic storage"},
		{"anonymous", `func box[T any](v T) any { return struct{ V T }{v} }`, "unresolved generic storage"},
		{"nested wrapper", `type Box[T any] struct { V []T }`, "unresolved generic storage"},
		{"native", `func native(v im.FrozenSlice[int])`, "unsupported function without a Go body"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if reports := checkSource(t, prefix+tc.source); !strings.Contains(reports, tc.want) {
				t.Fatalf("diagnostics = %q; want %q", reports, tc.want)
			}
		})
	}
}

// TestGenericPointerBoundary rejects protected generic reference construction.
func TestGenericPointerBoundary(t *testing.T) {
	const prefix = `type Value struct { im.Immut; N int }
 `
	cases := []string{
		`func box[T any](v T) any { return &v }; var _ = box(Value{})`,
		`func fresh[T any]() *T { return new(T) }; var _ = fresh[Value]()`,
		`func box[T comparable](v T) any { return &v }; var _ = box(Value{})`,
		`func head[T any](xs []T) any { return &xs[0] }; var _ = head([]Value{{}})`,
	}
	for _, source := range cases {
		t.Run(source[:min(len(source), 40)], func(t *testing.T) {
			if reports := checkSource(t, prefix+source); !strings.Contains(reports, "unsupported") {
				t.Fatalf("missing unsupported generic reference diagnostic: %s", reports)
			}
		})
	}
}

// TestSafeGenericStorage accepts bounds that exclude marked storage and ordinary interface copies.
func TestSafeGenericStorage(t *testing.T) {
	cases := []string{
		`func box[T interface{ ~int | ~string }](v T) any { return &v }; var _ = im.Immut{}`,
		`type Plain struct { X int }
   func box[T Plain](v T) any { return &v }; var _ = im.Immut{}`,
		`type Plain struct { X int }
   type Box[T Plain] struct { Value T }; var _ = im.Immut{}`,
		`type Scalar interface { ~int | ~string }
   type Smaller interface { Scalar; ~int }
   func box[T Smaller](v T) any { return &v }; var _ = im.Immut{}`,
	}
	for _, source := range cases {
		t.Run(source[:min(len(source), 40)], func(t *testing.T) {
			if reports := checkSource(t, source); reports != "" {
				t.Fatalf("unexpected diagnostics: %s", reports)
			}
		})
	}
}
