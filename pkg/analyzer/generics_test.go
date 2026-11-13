package analyzer

import (
	"strings"
	"testing"
)

// TestGenericAliasBoundaries checks that type erasure cannot hide pointers or unmarked owners.
func TestGenericAliasBoundaries(t *testing.T) {
	const prefix = `type Value struct { im.Marker; N int }
 `
	cases := []struct{ name, source, want string }{
		{"box", `func box[T any](v T) any { return &v }; var _ = box(Value{})`, "unsupported pointer"},
		{"head", `func head[T any](xs []T) any { return &xs[0] }; var _ = head([]Value{{}})`, "unsupported pointer"},
		{"new", `func box[T any]() any { return new(T) }; var _ = box[Value]()`, "unsupported pointer"},
		{"clear", `func clear[T any](v any) { var z T; *v.(*T) = z }`, "unsupported pointer"},
		{"nested", `func box[T any](v []T) any { return &v }`, "unsupported pointer"},
		{"comparable", `func box[T comparable](v T) any { return &v }`, "unsupported pointer"},
		{"method", `func box[T interface{ String() string }](v T) any { return &v }`, "unsupported pointer"},
		{"wrapper", `type Box[T any] struct { V T }
   func box[T any](v T) any { return Box[T]{V:v} }; var _ = box(Value{})`, "unresolved generic storage"},
		{"anonymous", `func box[T any](v T) any { return struct{ V T }{v} }`, "unresolved generic storage"},
		{"nested wrapper", `type Box[T any] struct { V []T }`, "unresolved generic storage"},
		{"native", `func native(v im.Slice[int])`, "unsupported function without a Go body"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if reports := checkSource(t, prefix+tc.source); !strings.Contains(reports, tc.want) {
				t.Fatalf("diagnostics = %q; want %q", reports, tc.want)
			}
		})
	}
}

// TestSafeGenericStorage accepts bounds that exclude marked storage and ordinary interface copies.
func TestSafeGenericStorage(t *testing.T) {
	cases := []string{
		`func box[T interface{ ~int | ~string }](v T) any { return &v }; var _ = im.Marker{}`,
		`type Plain struct { X int }
   func box[T Plain](v T) any { return &v }; var _ = im.Marker{}`,
		`type Plain struct { X int }
   type Box[T Plain] struct { Value T }; var _ = im.Marker{}`,
		`type Scalar interface { ~int | ~string }
   type Smaller interface { Scalar; ~int }
   func box[T Smaller](v T) any { return &v }; var _ = im.Marker{}`,
		`type Value struct { im.Marker; N int }
   type Box struct { V any }; var _ = Box{V: Value{}}`,
		`func copy[T any](v T) any { return v }; var _ = im.Marker{}`,
	}
	for _, source := range cases {
		t.Run(source[:min(len(source), 40)], func(t *testing.T) {
			if reports := checkSource(t, source); reports != "" {
				t.Fatalf("unexpected diagnostics: %s", reports)
			}
		})
	}
}
