//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Child struct {
	im.Frozen
	X, Y int
}
type Good struct {
	im.Frozen
	X, Y  int
	Child Child
	Array [2]int
}

func read(pointer *Good) int   { return pointer.X }
func readInt(pointer *int) int { return *pointer }

func bad() {
	var value Good
	pointer := new(value)
	value.X = 2
	_ = read(pointer)
}
