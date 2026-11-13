package docrejected

import "github.com/lucasavila00/immutago/pkg/im"

type Point struct {
	im.Immut
	X int
}

func implicitArrayIndex() int {
	values := [2]Point{}
	p := &values
	return p[0].X
}

func sharedArrayWrite() {
	values := [2]Point{}
	p := &values
	(*p)[0].X = 1
}

func liveArrayWrite() int {
	values := [2]Point{}
	p := &values
	values[0].X = 1
	return (*p)[0].X
}

type LocalPoint struct {
	im.ImmutNoRef
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
