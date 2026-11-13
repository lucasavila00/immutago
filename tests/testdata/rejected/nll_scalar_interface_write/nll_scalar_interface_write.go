package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	X int
}

func bad() {
	var value Good
	var boxed any = &value.X
	*boxed.(*int) = 2 // want "borrow|shared reference"
}
