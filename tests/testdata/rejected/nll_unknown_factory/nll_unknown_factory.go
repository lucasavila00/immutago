package rejected

import im "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	X int
}

func bad(factory func() *Good) {
	pointer := factory() // want "unsupported|shared reference"
	pointer.X = 2
}
