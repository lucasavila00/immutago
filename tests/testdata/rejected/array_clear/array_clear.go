package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Shared
	Array [2]int
}

func mutate() {
	var value Good
	pointer := new(value)
	clear(value.Array[:])
	_ = pointer.Array[0]
}
