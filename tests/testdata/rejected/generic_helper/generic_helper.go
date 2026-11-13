package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Marker
	Value int
	Array [2]int
}

func Build[T any](value T) im.FrozenSlice[T] { frozen := im.FreezeSlice([]T{value}); return frozen }
