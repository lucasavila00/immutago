package rejected

import im "github.com/lucasavila00/immutago/pkg/immutable"

type Good struct {
	im.Marker
	Value int
	Array [2]int
}

var Bad = im.NewMap(map[string][]int{"a": {1}}) // want "immutable collection: .*not provably immutable"
