package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Child struct {
	im.Shared
	X, Y int
}
type Good struct {
	im.Shared
	X, Y  int
	Child Child
	Array [2]int
}

func read(pointer *Good) int   { return pointer.X }
func readInt(pointer *int) int { return *pointer }

func bad(count int) {
	var value Good
	pointer := new(value)
	for index := 0; index < count; index++ {
		value.X = index
		_ = read(pointer)
	}
}
