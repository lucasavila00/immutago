package main

import "github.com/lucasavila00/immutago/pkg/im"

type Point struct {
	im.Shared
	X, Y int
}
type Shape struct {
	im.Shared
	Point Point
	Items im.FrozenSlice[int]
}

type PointArray struct {
	im.Shared
	Values [2]Point
}

// referenceForms checks fresh marked objects and value collection reads.
func referenceForms() {
	p := &PointArray{Values: [2]Point{{X: 1}}}
	x := p.Values[0].X
	next := *p
	next = PointArray{Values: [2]Point{{X: 2}}}
	child := &Point{X: 2}
	copied := *child
	var items im.FrozenSlice[int]
	count := items.Len()
	println(x, next.Values[0].X, copied.X, count)
}
