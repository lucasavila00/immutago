package main

import "github.com/lucasavila00/immutago/pkg/im"

type Point struct {
	im.Marker
	X, Y int
}
type Shape struct {
	im.Marker
	Point Point
	Items im.FrozenSlice[int]
}

// referenceForms checks whole arrays and protected fields from the rules.
func referenceForms() {
	values := [2]Point{}
	p := &values
	x := (*p)[0].X
	next := *p
	next[0].X = 1
	values[0].X = 2
	shape := Shape{}
	child := &shape.Point
	copied := *child
	items := &shape.Items
	count := items.Len()
	println(x, next[0].X, copied.X, count)
}
