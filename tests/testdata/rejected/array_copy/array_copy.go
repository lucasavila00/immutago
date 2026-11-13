package rejected

import im "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	Array [2]int
}

func mutate() {
	var value Good
	pointer := &value
	copy(value.Array[:], []int{1}) // want "borrow|shared reference"
	_ = pointer.Array[0]
}
