package rejected

import im "github.com/lucasavila00/immutago/pkg/im"

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

func bad(count int) {
	var value Good
	pointer := &value
	for index := 0; index < count; index++ {
		value.X = index // want "borrow|shared reference"
		_ = read(pointer)
	}
}
