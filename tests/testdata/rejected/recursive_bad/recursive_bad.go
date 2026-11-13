package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	Value int
	Array [2]int
}

type Bad struct { // want "immutable field Children: .*not provably immutable"
	im.Marker
	Children im.Slice[Bad]
	Mutable  []int
}
