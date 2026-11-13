package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	Value int
	Array [2]int
}

type Bad[T ~int | ~[]int] struct {
	im.Marker
	Value T
}
