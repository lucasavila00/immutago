package rejected

import im "github.com/lucasavila00/immutago/pkg/immutable"

type Good struct {
	im.Marker
	Value int
	Array [2]int
}

var Bad im.Slice[im.Map[string, []int]] // want "immutable collection: .*not provably immutable"
