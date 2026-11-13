package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Child struct {
	im.Immut
	X, Y int
}
type Good struct {
	im.Immut
	X, Y  int
	Child Child
	Array [2]int
}

func read(pointer *Good) int   { return pointer.X }
func readInt(pointer *int) int { return *pointer }

func bad(condition bool) {
	var first Good
	var second Good
	var pointer *Good
	if condition {
		pointer = &first
	} else {
		pointer = &second
	}
	first.X = 2
	_ = read(pointer)
}
