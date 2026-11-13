package objectcases

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

func read(pointer *Good) int { return pointer.X }
func Conditional(condition bool) Good {
	x := 2
	if condition {
		x = 1
	}
	x += 3
	return Good{X: x, Y: 1}
}
func Shared() *Good {
	value := &Good{X: 1}
	copy := value
	value = &Good{X: 2}
	_ = read(value)
	return copy
}
func ValueCopies() Good {
	value := Good{X: 1}
	copied := value
	copied = Good{X: 2}
	return copied
}
