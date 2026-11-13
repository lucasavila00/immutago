package rejected

import im "github.com/lucasavila00/immutago/pkg/immutable"

type Good struct {
	im.Marker
	Value int
	Array [2]int
}

type Bad struct { // want "immutable.Marker must be embedded by value"
	Marker im.Marker
	Value  int
}
