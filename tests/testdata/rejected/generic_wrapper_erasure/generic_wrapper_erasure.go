package rejected

import im "github.com/lucasavila00/immutago/pkg/immutable"

type Good struct {
	im.Marker
	Value int
}

type Box[T any] struct{ Value T } // want "unsupported unmarked field Value: unresolved generic storage"
func wrap[T any](value T) any     { return Box[T]{Value: value} }

var Hidden = wrap(Good{Value: 1})
