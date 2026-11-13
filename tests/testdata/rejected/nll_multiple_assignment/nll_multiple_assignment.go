package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Child struct {
	im.Marker
	X, Y int
}
type Good struct {
	im.Marker
	X, Y  int
	Child Child
	Array [2]int
}

func read(pointer *Good) int   { return pointer.X }
func readInt(pointer *int) int { return *pointer }

func bad() {
	var value Good
	var pointer *Good
	pointer, value.X = &value, 2
	_ = read(pointer)
}
