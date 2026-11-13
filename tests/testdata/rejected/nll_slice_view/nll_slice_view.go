package rejected

import im "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	Array [2]int
}

func bad() {
	var value Good
	view := value.Array[:]
	value.Array[0] = 2 // want "borrow|shared reference"
	_ = view[0]
}
