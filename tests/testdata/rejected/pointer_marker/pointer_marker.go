package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	Value int
	Array [2]int
}

type Bad struct { // want "im.Marker must be embedded by value"
	*im.Marker
	Value int
}
