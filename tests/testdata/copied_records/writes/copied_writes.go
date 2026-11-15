package writes

import (
	"example.com/consumer/api"
	"example.com/consumer/frozen"
	"github.com/lucasavila00/immutago/pkg/im"
)

type Holder struct {
	Point api.Point
	Other int
}
type Result struct {
	im.Local
	Point api.Point
}

func Writes(point api.Point, summary api.Summary, stored frozen.Snapshot, local Result) {
	point.X++
	summary.Points[0][0] = api.Point{}
	state := Holder{Point: point}
	state.Point.X = 2
	values := []api.Point{point}
	values[0].X++
	array := [2]api.Point{point}
	array[0].Y = 2
	stored.Point = point
	local.Point.X = 3
}

func Addresses(point api.Point, holder Holder) {
	_ = &point
	_ = &point.X
	_ = &holder.Point
	_ = new(api.Point)
	_ = &api.Point{}
}

type Alias = *api.Point
type Defined *api.Point

var pointer Alias
var defined Defined
var zero **api.Point

type LocalPoint struct {
	im.Copied
	N int
}

func (p *LocalPoint) Change() { p.N++ }

func BuilderAliases() {
	values := []api.Point{{}}
	alias := values
	_ = alias
	_ = im.FreezeSlice(values)
}

func Consumed() {
	values := []api.Point{{}}
	_ = im.FreezeSlice(values)
	values[0] = api.Point{}
}

func ProjectionPointers(value im.Projection[api.Point], values im.SliceProjection[api.Point]) {
	_ = im.CopyRecord(value)
	_ = values.AtPointer(0)
}
