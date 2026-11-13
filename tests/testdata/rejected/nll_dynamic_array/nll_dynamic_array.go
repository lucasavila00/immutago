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

func read(pointer *Good) int        { return pointer.X }
func readInt(pointer *int) int      { return *pointer }
func identity[T any](pointer *T) *T { return pointer }
func (value *Good) Read() int       { return value.X }

func bad(first, second int) {
	var value Good
	pointer := &value.Array[first]
	value.Array[second] = 2 // want "borrow|shared reference"
	_ = readInt(pointer)
}
