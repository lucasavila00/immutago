package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Immut
	Value int
	Array [2]int
}

type Bad[T ~int | ~[]int] struct {
	im.Immut
	Value T
}
