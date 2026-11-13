package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	Value int
	Array [2]int
}

type Bad struct { // want "immutable field Value: struct does not embed"
	im.Marker
	Value struct{ Count int }
}
