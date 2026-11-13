package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	Value int
	Array [2]int
}

var Bad im.Map[string, *int] // want "immutable collection: .*not provably immutable"
