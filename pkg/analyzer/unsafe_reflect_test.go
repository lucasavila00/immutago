package analyzer

import (
	"strings"
	"testing"
)

// TestOrdinaryUnsafeReflection keeps unrelated operations valid in a checked package.
func TestOrdinaryUnsafeReflection(t *testing.T) {
	cases := []struct{ name, source string }{
		{"unsafe pointers", `import "unsafe"
type Value struct { im.Frozen; N int }
func ordinary(p *int) unsafe.Pointer { return unsafe.Pointer(p) }`},
		{"unsafe string", `import "unsafe"
type Value struct { im.Frozen; N int }
func ordinary(s string) []byte { return unsafe.Slice(unsafe.StringData(s), len(s)) }`},
		{"raw import", "import `unsafe`\ntype Value struct { im.Frozen }; var _ unsafe.Pointer"},
		{"reflection", `import "reflect"
type Value struct { im.Frozen; N int }
func ordinary(p *int) { reflect.ValueOf(p).Elem().SetInt(2) }`},
		{"reflection pointer", `import "reflect"
type Value struct { im.Frozen; N int }
func ordinary(v reflect.Value) { _ = v.UnsafePointer() }`},
		{"reflection type", `import "reflect"
type Value struct { im.Frozen; N int }
var ordinary = reflect.TypeFor[int]()`},
		{"scalar read", `import "unsafe"
type Value struct { im.Frozen; Name string }
func ordinary(v Value) *byte { return unsafe.StringData(v.Name) }`},
		{"copied string", `import "unsafe"
type Value struct { im.Frozen; Name string }
func ordinary(v Value) *byte { s := v.Name; return unsafe.StringData(s) }`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if reports := checkSource(t, tc.source); reports != "" {
				t.Fatalf("ordinary operation rejected: %s", reports)
			}
		})
	}
}

// TestProtectedUnsafeReflection keeps protected transport inside the existing forms.
func TestProtectedUnsafeReflection(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"unsafe pointer", `import "unsafe"
type Value struct { im.Frozen; N int }
//nolint:immutago
func f(p *Value) { _ = unsafe.Pointer(p) }`, "protected value conversions"},
		{"unsafe result", `import "unsafe"
type Value struct { im.Frozen; N int }
func f(p unsafe.Pointer) *Value { return (*Value)(p) }`, "shared pointer conversions"},
		{"unsafe field", `import "unsafe"
type Value struct { im.Frozen; N int }
func f(v Value) { _ = unsafe.Pointer(&v.N) }`, "interior addresses of protected storage"},
		{"reflection value", `import r "reflect"
type Value struct { im.Frozen; N int }
func f(p *Value) { r.ValueOf(p).Elem().FieldByName("N").SetInt(1) }`, "boxing immutable storage"},
		{"reflection type", "import `reflect`\n" + `type Value struct { im.Local; N int }
func f(v Value) { _ = reflect.TypeOf(v) }`, "boxing immutable storage"},
		{"reflection callback", `import "reflect"
type Value struct { im.Frozen; N int }
func f(p *Value) { inspect := reflect.ValueOf; _ = inspect(p) }`, "boxing immutable storage"},
		{"reflection generic", `import "reflect"
type Value struct { im.Local; N int }
var value = reflect.TypeFor[Value]()`, "user generic instantiations"},
		{"reflection container", `import "reflect"
type Value struct { im.Frozen; N int }
func f(values []Value) { reflect.ValueOf(values).Index(0).FieldByName("N").SetInt(1) }`, "boxing immutable storage"},
		{"frozen reflection", `import "reflect"
func f(values im.SliceProjection[int]) { _ = reflect.ValueOf(values) }`, "boxing immutable storage"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if reports := checkSource(t, tc.source); !strings.Contains(reports, tc.want) {
				t.Fatalf("diagnostics = %q; want %q", reports, tc.want)
			}
		})
	}
}
