//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Frozen
	Value int
}

type Box[T any] struct{ Value T }

func wrap[T any](value T) any { return Box[T]{Value: value} }

var Hidden = wrap(Good{Value: 1})
