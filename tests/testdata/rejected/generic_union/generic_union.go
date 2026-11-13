package rejected

import im "github.com/lucasavila00/immutago/pkg/immutable"

type Good struct {
	im.Marker
	Value int
	Array [2]int
}

type Bad[T ~int | ~[]int] struct { // want "immutable field Value: type parameter constraint"
	im.Marker
	Value T
}
