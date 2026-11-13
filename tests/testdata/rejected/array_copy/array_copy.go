package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Immut
	Array [2]int
}

func mutate() {
	var value Good
	pointer := new(value)
	copy(value.Array[:], []int{1})
	_ = pointer.Array[0]
}
