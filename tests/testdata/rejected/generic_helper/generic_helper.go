package rejected

import im "github.com/lucasavila00/immutago/pkg/immutable"

type Good struct {
	im.Marker
	Value int
	Array [2]int
}

func Build[T any](value T) im.Slice[T] { return im.NewSlice(value) } // want "immutable collection: .*type parameter constraint"
