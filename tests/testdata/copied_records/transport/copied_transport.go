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
