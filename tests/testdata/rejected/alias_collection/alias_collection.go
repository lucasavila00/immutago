package rejected

import im "github.com/lucasavila00/immutago/pkg/immutable"

type Good struct {
	im.Marker
	Value int
	Array [2]int
}

type Bad = im.Slice[[]int] // want "immutable collection: .*not provably immutable"
