package docrejected

import "github.com/lucasavila00/immutago/pkg/im"

type Point struct {
	im.Shared
	X int
}

type PointArray struct {
	im.Shared
	Values [2]Point
}

func arrayReference() { p := &[2]Point{}; _ = p }
func sharedArrayWrite() {
	p := &PointArray{}
	p.Values[0].X = 1
}
func localArrayWrite() {
	values := [2]Point{}
	values[0].X = 1
}

type LocalPoint struct {
	im.LocalRef
	X int
}

func ownerCapture() int {
	point := &LocalPoint{}
	callback := func() int { return point.X }
	return callback()
}

func identity[T any](value T) T { return value }

func genericValue() Point {
	point := Point{}
	return identity(point)
}

func makePair() (Point, bool) { return Point{}, true }

func bindPair() {
	point, found := makePair()
	println(point.X, found)
}
