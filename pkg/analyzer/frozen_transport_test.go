package analyzer

import (
	"strings"
	"testing"
)

// TestFrozenValueTransport checks ordinary uses of Frozen values.
func TestFrozenValueTransport(t *testing.T) {
	const prefix = `type Data struct { im.Frozen; Text im.FrozenString; Values [2]im.FrozenString }
type Alias = Data
type Reader interface { im.FrozenInterface; Len() int }
func identity[T any](value T) T { return value }
func (value Data) Len() int { return value.Text.Len() }
`
	for name, source := range map[string]string{
		"ordinary interface": `type Factory interface{Make()Data};func use(factory Factory)Data{return factory.Make()}`,
		"pointer generic":    `func use(value *Data) { _ = identity(value); _ = func() int { return value.Len() } }`,
		"interface":          `func use(s im.FrozenString) { var boxed any = s; _ = boxed.(im.FrozenString) }`,
		"generic":            `func use(s im.FrozenString) { _ = identity(s) }`,
		"generic body": `func use[T any](value T, s im.FrozenString) T { var boxed any = s; _ = boxed;
   return value }`,
		"capture":  `func use(s im.FrozenString) func() int { return func() int { return s.Len() } }`,
		"callback": `func use(s im.FrozenString, call func(im.FrozenString) im.FrozenString) { _ = call(s) }`,
		"nested": `func use(value Alias) { var boxed any = value; _ = boxed; _ = identity(value.Values);
   _ = func() int { return value.Text.Len() } }`,
		"method value":      `func use(value Data) { read := value.Len; _ = read() }`,
		"method expression": `func use(value Data) { read := Data.Len; _ = read(value) }`,
		"collection methods": `func use(s im.FrozenString) { read := s.Len; _ = read(); size := im.FrozenString.Len;
   _ = size(s) }`,
		"tuple callback":       `func use(call func()(Data, bool)) { value, ok := call(); _, _ = value, ok }`,
		"multiple values":      `func use(a, b Data) { a, b = b, a; var first, second = a, b; _, _ = first, second }`,
		"interface constraint": `func use[T Reader](value T) int { return value.Len() }`,
		"generic constraint":   `func use[T Data](value T) any { return identity(value) }`,
		"generic closure": `func use[T any](value T, s im.FrozenString) func() T { return func() T { _ = s.Len();
   return value } }`,
		"go and defer": `func read(Data){};func use(value Data) { go read(value); defer read(value);
   go value.Len(); defer value.Len() }`,
	} {
		t.Run(name, func(t *testing.T) {
			if got := checkSource(t, prefix+source); got != "" {
				t.Fatal(got)
			}
		})
	}
}

// TestFrozenTransportProtection keeps Local movement and protected targets restricted.
func TestFrozenTransportProtection(t *testing.T) {
	const prefix = `type Data struct { im.Frozen; N int }
type View struct { im.Local; Text im.StringProjection }
type Alias = View
type Contract interface { im.FrozenInterface; Number() int }
func (value Data) Number() int { return value.N }
func (value *Data) Read() int { return value.N }
func identity[T any](value T) T { return value }
`
	for name, test := range map[string]struct{ source, want string }{
		"unmarked generic": {`type Box[T any]struct{ Value T };var _ Box[Data]`, "unmarked struct"},
		"local boxing":     {`func use(value Alias) { var boxed any = value; _ = boxed }`, "ordinary interfaces require"},
		"local generic":    {`func use(value View) { _ = identity(value) }`, "user generic instantiations"},
		"local capture": {`func use(value View) { _ = func() int { return value.Text.Len() } }`,
			"closures require"},
		"nested capture": {`func use(values [2]View) { _ = func() int { return values[0].Text.Len() } }`,
			"closures require"},
		"local callback":   {`func use(value View, call func(View)) { call(value) }`, "callbacks require"},
		"local constraint": {`func use[T View](value T) any { return value }`, "user generic declarations"},
		"write":            {`func use(value Data) { value.N = 2 }`, "writes to immutable"},
		"capture write":    {`func use(value Data) { _ = func() { value.N = 2 } }`, "writes to immutable"},
		"address":          {`func use(value Data) { _ = &value.N }`, "interior addresses"},
		"implicit address": {`func use(value Data) { _ = value.Read() }`, "receivers must match"},
		"generic implementation": {`func use[T Contract](value T) Contract { return value }`,
			"Frozen interface implementation"},
		"assert implementation": {`func use(value any) { _ = value.(Contract) }`, "Frozen interface implementation"},
		"tuple assertion": {`func use(value any) { result, ok := value.(Contract); _ = result;
   _ = ok }`, "Frozen interface implementation"},
		"pointer boxing": {`func use(value *View) { var boxed any = value; _ = boxed }`, "ordinary interfaces require"},
		"builder alias":  {`func use() { values := []Data{}; alias := values; _ = alias }`, "alias"},
	} {
		t.Run(name, func(t *testing.T) {
			if got := checkSource(t, prefix+test.source); !strings.Contains(got, test.want) {
				t.Fatalf("got %q; want %q", got, test.want)
			}
		})
	}
}
