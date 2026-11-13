package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	X int
}

func read(pointer *Good) int { return pointer.X }
func bad() {
	owner := &Good{}
	shared := owner
	owner.X = 2 // want "borrow|shared reference"
	_ = read(shared)
}
