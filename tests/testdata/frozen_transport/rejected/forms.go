package rejected

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

type View struct {
	im.Local
	Text im.StringProjection
}
type Alias = View
type Nested struct {
	im.Local
	Views [2]View
}

func localValues(value Alias, nested Nested) {
	var boxed any = value
	_ = boxed
	_ = api.Identity(value)
	_ = func() int { return value.Text.Len() }
	_ = func() int { return nested.Views[0].Text.Len() }
	_ = []any{value}
}
func localCallback(call func(im.StringProjection), value im.StringProjection) { call(value) }
func localGeneric[T View](value T) any                                        { return value }
func localBody[T any](value T) {
	local := im.ProjectString("x")
	var boxed any = local
	_ = boxed
}
func localBindings(a, b View) { a, b = b, a }
func localConstant(value *View) func() int {
	return func() int { return len([1]*View{value}) }
}
func writes(value api.Data) {
	value.Counts[0].N = 3
	_ = &value.Text
	_ = func() { value.Counts[0].N++ }
	_ = value.Counts[0].Read()
}
func builders() {
	values := []im.FrozenString{im.FreezeString("x")}
	alias := values
	_ = alias
}
func consumed() {
	values := []int{1}
	frozen := im.FreezeSlice(values)
	_ = frozen
	values[0] = 2
}
func pointer(value *api.Count) {
	var boxed any = value
	_ = boxed
}

type Box[T any] struct{ Value T }

var _ Box[im.FrozenString]
