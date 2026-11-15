package analyzer

import "testing"

// TestFrozenValueTransport checks ordinary uses of persistent Frozen values.
func TestFrozenValueTransport(t *testing.T) {
	const prefix = `package sample
import "github.com/lucasavila00/immutago/pkg/im"
type Data struct { im.Frozen; Text im.FrozenString; Values [2]im.FrozenString }
type Alias = Data
func identity[T any](value T) T { return value }
func (value Data) Len() int { return value.Text.Len() }
`
	for name, source := range map[string]string{
		"interface": `func use(s im.FrozenString) { var boxed any = s; _ = boxed.(im.FrozenString) }`,
		"generic": `func use(s im.FrozenString) { _ = identity(s) }`,
		"generic body": `func use[T any](value T, s im.FrozenString) T { var boxed any = s; _ = boxed; return value }`,
		"capture": `func use(s im.FrozenString) func() int { return func() int { return s.Len() } }`,
		"callback": `func use(s im.FrozenString, call func(im.FrozenString) im.FrozenString) { _ = call(s) }`,
		"nested": `func use(value Alias) { var boxed any = value; _ = identity(value.Values); _ = func() int { return value.Text.Len() } }`,
		"method value": `func use(value Data) { read := value.Len; _ = read() }`,
		"method expression": `func use(value Data) { read := Data.Len; _ = read(value) }`,
	} {
		t.Run(name, func(t *testing.T) {
			if got := checkSource(t, prefix+source); got != "" {
				t.Fatal(got)
			}
		})
	}
}
