package main

import "github.com/lucasavila00/immutago/pkg/im"

type Point struct {
	im.Immut
	X, Y int
}
type Shape struct {
	im.Immut
	Point Point
	Items im.FrozenSlice[int]
}

// referenceForms checks fresh objects, arrays, and shared fields from the rules.
func referenceForms() {
	p := &[2]Point{}
	x := (*p)[0].X
	next := *p
	next = [2]Point{{X: 1}}
	child := &Point{X: 2}
	copied := *child
	items := new(im.FrozenSlice[int])
	count := items.Len()
	println(x, next[0].X, copied.X, count)
}
