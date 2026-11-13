package rejected

import im "github.com/lucasavila00/immutago/pkg/immutable"

type Good struct {
	im.Marker
	Value int
	Array [2]int
}

func mutate(value *Good) {} // want "unsupported pointer to immutable storage"
