package transport

import "example.com/consumer/api"

type Holder struct{ Point api.Point }
type Embedded struct{ api.Point }

func read(value api.Point) int                    { return value.X }
func identity[T any](value T) T                   { return value }
func boxed(value any)                             {}
func boxedMany(values ...any)                     {}
func pair(value api.Point) (api.Point, api.Point) { return value, value }

func Uses(point api.Point, holder Holder, values []api.Point) {
	var stored any = point
	_ = stored
	boxed(&holder)
	boxedMany(point)
	_ = identity(values)
	callback := read
	_ = callback(point)
	_ = func() int { return point.X }
	_ = func() int { return len([1]api.Point{point}) }
	go read(point)
	defer read(point)
	first, second := pair(point)
	_, _ = first, second
}

func Promotion(value Embedded) int     { return value.Sum() }
func MethodValue(value api.Point)      { method := value.Sum; _ = method }
func MethodExpression(value api.Point) { _ = api.Point.Sum(value) }
func Channel()                         { var values chan api.Point; _ = values }

type Pointer = *api.Point
type Defined *api.Point

var pointer Pointer
var defined Defined
var zero **api.Point

func Pointers(value api.Point) *api.Point {
	p := &value
	p.Set(2)
	*(&p.X) = 3
	var nilPointer Pointer = nil
	var converted Defined = nilPointer
	_, _, _ = converted, new(api.Point), &api.Point{}
	return p
}

func TypeSwitch(value any) int {
	switch point := value.(type) {
	case *api.Point:
		point.X++
		return point.X
	case api.Point:
		point.X++
		return point.X
	}
	return 0
}

func CapturePointer(point *api.Point) func() {
	return func() { point.X++ }
}
