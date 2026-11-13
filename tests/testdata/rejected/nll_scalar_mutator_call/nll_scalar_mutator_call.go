package rejected

import im "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	X int
}

func read(pointer *Good) int { return pointer.X }
func set(pointer *int)       { *pointer = 2 }
func bad() {
	var value Good
	set(&value.X) // want "borrow|shared reference"
}
