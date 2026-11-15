package accepted

import (
	"encoding"
	"encoding/json"

	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

type Payload struct {
	ID    string
	Ratio api.Ratio
	Rows  [2]api.Summary
}

type request struct {
	id     string
	limits api.Point
	a, b   int
}

func (r *request) release() { r.a++ }

func handle(r *request, lookup func(string) (float64, bool)) {
	defer r.release()
	x, y := r.a, r.b
	v, ok := lookup(r.id)
	r.a, r.b = 1, 2
	_, _, _, _ = x, y, v, ok
}

func (p *Payload) Decode(data []byte) error { return (p.Ratio).UnmarshalJSON(data) }

func Decode(data []byte) (Payload, error) {
	var value Payload
	err := json.Unmarshal(data, &value)
	return value, err
}

func DecodeText(data []byte) (api.Ratio, error) {
	var value api.Ratio
	var decoder encoding.TextUnmarshaler = &value
	err := decoder.UnmarshalText(data)
	return value, err
}

func Mutate(value api.Point) (api.Point, api.Point) {
	copy := value
	copy.X++
	pointer := &copy
	pointer.Set(9)
	*pointer = api.Point{X: copy.X, Y: 4}
	return value, copy
}

func Nested(value *Payload) {
	value.Rows[0].Points[1][0].Set(3)
	value.Rows[1].Counts[0]++
	value.Rows[0].Points[0][0].X = 4
	pointer := &value.Rows[0].Points[0][0]
	pointer.Y++
}

func MutableBuilders() im.FrozenSlice[api.Point] {
	values := []api.Point{{}}
	values[0].X++
	values[0].Y = 3
	frozen := im.FreezeSlice(values)
	return frozen
}

func Methods(point api.Point) (int, int) {
	x, y := point.Values()
	point.X, point.Y = y, x
	set := point.Set
	set(3)
	(*api.Point).Set(&point, 4)
	return point.Values()
}

func Concurrent(point api.Point, done chan<- api.Point) {
	defer func() { done <- point }()
	go func(value api.Point) { value.Set(7); done <- value }(point)
}

func Channels(point api.Point) api.Point {
	values := make(chan api.Point, 1)
	values <- point
	value, ok := <-values
	if !ok {
		return api.Point{}
	}
	return value
}
